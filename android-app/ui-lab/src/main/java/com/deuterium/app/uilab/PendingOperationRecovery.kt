package com.deuterium.app.uilab

import org.json.JSONObject
import java.net.URLEncoder

internal data class FinancialScope(val owner: String, val origin: String) {
    fun verify(pending: JSONObject) {
        if (owner.isBlank() || pending.optString("owner") != owner || pending.optString("origin") != origin)
            throw ApiFailure("REQUEST_SCOPE_MISMATCH", "这笔交易与当前账号不匹配，请联系平台协助处理")
    }

    fun verifyCurrent(current: FinancialScope) {
        if (owner.isBlank() || this != current)
            throw ApiFailure("SESSION_CHANGED", "账号或服务器已切换，请返回原账号查看交易进度")
    }
}

/** Only an authoritative absent lookup permits the original command to be repeated. Never invent a new intent. */
internal suspend fun recoverPendingOperation(
    kind: String,
    pending: JSONObject,
    scope: FinancialScope,
    currentScope: () -> FinancialScope,
    request: suspend (String, String, JSONObject?) -> JSONObject,
    persist: (JSONObject?) -> Unit
): JSONObject {
    scope.verify(pending)
    scope.verifyCurrent(currentScope())
    val original = JSONObject(pending.getJSONObject("request").toString())
    val operationId = pending.optString("operationId").takeUnless { it.isBlank() || it == "null" }
    val operation = if (operationId != null) {
        request("GET", "/operations/$operationId", null)
    } else {
        val key = URLEncoder.encode(original.getString("clientRequestId"), "UTF-8")
        try {
            request("GET", "/operations/by-client-request?clientRequestId=$key&kind=$kind", null)
        } catch (failure: ApiFailure) {
            if (failure.status != 404 || failure.code != "NOT_FOUND") throw failure
            scope.verifyCurrent(currentScope())
            val path = when (kind) {
                "STORE_PURCHASE" -> "/store/orders"
                "MARKET_PURCHASE" -> "/market/orders"
                "AI_PURCHASE" -> "/ai/purchases"
                "COMMISSION_PUBLISH" -> "/commissions"
                else -> throw ApiFailure("UNSUPPORTED_RECOVERY", "此操作需要按原记录核对")
            }
            try {
                request("POST", path, original).getJSONObject("operation")
            } catch (rejection: ApiFailure) {
                // An expired original quote cannot be replaced or paid silently. Other errors retain the original intent.
                if ((rejection.status == 409 && rejection.code == "QUOTE_EXPIRED") || (kind == "AI_PURCHASE" && rejection.code in setOf("AI_PLAN_CHANGED","AI_PLAN_UNAVAILABLE","AI_PURCHASE_UNAVAILABLE","AI_PLAN_ACTIVE","AI_DOWNGRADE_NOT_ALLOWED","AI_UPGRADE_UNAVAILABLE","AI_QUOTE_REQUIRED","AI_QUOTE_CHANGED","QUOTE_EXPIRED","QUOTE_ALREADY_USED","AMOUNT_LIMIT","CAPABILITY_UNAVAILABLE"))) {
                    scope.verifyCurrent(currentScope())
                    persist(null)
                }
                throw rejection
            }
        }
    }
    scope.verifyCurrent(currentScope())
    persist(JSONObject(pending.toString()).put("operationId", operation.getString("operationId")))
    return operation
}
