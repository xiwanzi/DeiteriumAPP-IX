async (page) => {
  const origin = "http://127.0.0.1:5219", passed = [], writes = [], errors = [];
  const check = (value, label) => { if (!value) throw new Error(label); passed.push(label); };
  let value = { caseId: "case-one", transactionKind: "ORDER", transactionId: "order-case", status: "SUBMITTED", description: "买家申请核对已经交付的内容", desiredResolution: "FULL_REFUND", requestedRefundAmount: "50.00", applicant: { displayName: "买方", playerRef: "buyer" }, respondent: { displayName: "卖方", playerRef: "seller" }, fundsHeldForReview: true, assignedAdminRef: null, resolution: "", createdAt: "2026-09-08T03:00:00Z", updatedAt: "2026-09-08T03:00:00Z", version: 1, evidenceAssetIds: [], snapshot: { capturedAt: "2026-09-08T03:00:00Z", order: { amount: "50.00", fundsStatus: "HELD", items: [{ productId: "case-item", title: "交易快照商品", quantity: 1, unitPrice: "50.00" }] }, eventLog: [] } };
  page.on("pageerror", (error) => errors.push(error.message)); await page.unroute("**/api/v1/**");
  await page.route("**/api/v1/**", async (route) => {
    const request = route.request(), path = "/api/v1" + request.url().split("/api/v1")[1].split("?")[0]; const json = (data) => route.fulfill({ status: 200, contentType: "application/json", body: JSON.stringify({ data, page: { nextCursor: null } }) });
    if (path === "/api/v1/web/session") return json({ user: { userId: "case-admin", playerRef: "case-admin-ref", gameId: "CaseAdmin", qq: "100001", permissions: ["intervention.manage"] }, csrfToken: "case-csrf", expiresAt: "2030-01-01T00:00:00Z" });
    if (path === "/api/v1/admin/interventions") return json([value]);
    if (path === "/api/v1/admin/interventions/case-one") return json(value);
    if (path.startsWith("/api/v1/admin/interventions/case-one/")) {
      const body = request.postDataJSON(); writes.push({ path, body }); check(request.headers()["x-csrf-token"] === "case-csrf", "case mutation carries CSRF");
      if (path.endsWith("/assign")) value = { ...value, version: 2, assignedAdminRef: "case-admin-ref", status: "IN_REVIEW" };
      if (path.endsWith("/request-evidence")) value = { ...value, version: 3, status: "WAITING_EVIDENCE" };
      if (path.endsWith("/resolve")) value = { ...value, version: 4, status: "RESOLVING", resolution: body.reason };
      return json(value);
    }
    return route.fulfill({ status: 404, contentType: "application/json", body: JSON.stringify({ error: { code: "NOT_FOUND", message: "未找到" } }) });
  });
  await page.goto(origin + "/admin"); await page.getByRole("button", { name: /买方 与 卖方/ }).click(); await page.getByRole("button", { name: "由我处理", exact: true }).click(); await page.getByRole("dialog").last().getByRole("button", { name: "确认提交", exact: true }).click(); await page.getByRole("dialog").getByText("处理中", { exact: true }).waitFor();
  check(Object.keys(writes[0].body).sort().join(",") === "clientRequestId,expectedVersion" && writes[0].body.expectedVersion === 1, "assignment uses the current account and cannot nominate an arbitrary actor");
  await page.getByRole("button", { name: "要求补充资料", exact: true }).click(); await page.getByRole("textbox", { name: "说明", exact: true }).fill("请提供与交付约定相关的资料"); await page.getByRole("dialog").last().getByRole("button", { name: "确认提交", exact: true }).click(); await page.getByRole("dialog").getByText("待补充资料", { exact: true }).waitFor();
  check(writes[1].body.expectedVersion === 2 && writes[1].body.reason === "请提供与交付约定相关的资料", "evidence request uses the updated case version and explicit reason");
  await page.getByRole("button", { name: "提交处理决定", exact: true }).click(); await page.getByRole("combobox", { name: "处理决定", exact: true }).selectOption("PARTIAL_REFUND"); await page.getByRole("textbox", { name: "退款金额", exact: true }).fill("10.00"); await page.getByRole("textbox", { name: "处理依据", exact: true }).fill("依据双方已提交的交付证据，确认部分退款。"); await page.getByRole("dialog").last().getByRole("button", { name: "确认提交", exact: true }).click(); await page.getByRole("dialog").getByText("资金执行中", { exact: true }).waitFor();
  check(writes[2].body.expectedVersion === 3 && writes[2].body.refundAmount === "10.00" && writes[2].body.decision === "PARTIAL_REFUND", "resolution preserves decimal amount and current case version");
  check(await page.getByRole("button", { name: "提交处理决定", exact: true }).isDisabled(), "resolving financial action cannot be submitted again");
  value = { ...value, status: "RESOLVED", version: 5 }; await page.getByRole("button", { name: "刷新记录", exact: true }).click(); await page.getByRole("dialog").getByText("已处理", { exact: true }).waitFor();
  check(await page.getByRole("button", { name: "提交处理决定", exact: true }).count() === 0, "closed case keeps its result without offering another resolution");
  check(writes.length === 3, "polling and refresh create no extra case mutations");
  check(errors.length === 0, "case manager has no JavaScript runtime exception");
  return { mode: "isolated platform intervention contract responses; no real resolutions", count: passed.length, passed, errors };
}
