package com.deuterium.app.network

import com.deuterium.app.data.ApiResponse
import com.deuterium.app.data.AppUpdateCheckData
import com.deuterium.app.data.AiConversationResetData
import com.deuterium.app.data.AiConversationResetRequest
import com.deuterium.app.data.AiMeData
import com.deuterium.app.data.AiMessagesData
import com.deuterium.app.data.AiPlansData
import com.deuterium.app.data.AiPurchaseData
import com.deuterium.app.data.AiPurchaseRequest
import com.deuterium.app.data.AuthData
import com.deuterium.app.data.ChatMessagesData
import com.deuterium.app.data.CreateTransferRequest
import com.deuterium.app.data.LoginRequest
import com.deuterium.app.data.LogoutData
import com.deuterium.app.data.OnlinePlayersData
import com.deuterium.app.data.PasswordResetCodeRequest
import com.deuterium.app.data.PasswordResetData
import com.deuterium.app.data.PasswordResetRequest
import com.deuterium.app.data.PlayerDirectoryData
import com.deuterium.app.data.PlayerFollowData
import com.deuterium.app.data.PlayerFollowRequest
import com.deuterium.app.data.PresenceData
import com.deuterium.app.data.RecipientSearchData
import com.deuterium.app.data.RegisterRequest
import com.deuterium.app.data.RegistrationCodeRequest
import com.deuterium.app.data.TransferData
import com.deuterium.app.data.UserProfileData
import com.deuterium.app.data.VerificationTokenData
import com.deuterium.app.data.WalletBalanceData
import com.deuterium.app.data.WalletRecordsData
import retrofit2.Response
import retrofit2.http.Body
import retrofit2.http.DELETE
import retrofit2.http.GET
import retrofit2.http.POST
import retrofit2.http.Path
import retrofit2.http.Query

interface BackendApi {
    @POST("account/registration-code")
    suspend fun requestRegistrationCode(@Body body: RegistrationCodeRequest): Response<ApiResponse<VerificationTokenData>>

    @POST("account/register")
    suspend fun register(@Body body: RegisterRequest): Response<ApiResponse<AuthData>>

    @POST("account/login")
    suspend fun login(@Body body: LoginRequest): Response<ApiResponse<AuthData>>

    @POST("account/password-reset-code")
    suspend fun requestPasswordResetCode(@Body body: PasswordResetCodeRequest): Response<ApiResponse<VerificationTokenData>>

    @POST("account/password-reset")
    suspend fun resetPassword(@Body body: PasswordResetRequest): Response<ApiResponse<PasswordResetData>>

    @POST("account/logout")
    suspend fun logout(): Response<ApiResponse<LogoutData>>

    @GET("account/me")
    suspend fun me(): Response<ApiResponse<UserProfileData>>

    @GET("app/update-check")
    suspend fun updateCheck(
        @Query("versionCode") versionCode: Int,
        @Query("versionName") versionName: String
    ): Response<ApiResponse<AppUpdateCheckData>>

    @GET("wallet/balance")
    suspend fun walletBalance(): Response<ApiResponse<WalletBalanceData>>

    @POST("wallet/balance/refresh")
    suspend fun refreshWalletBalance(): Response<ApiResponse<WalletBalanceData>>

    @GET("wallet/recipients/search")
    suspend fun searchRecipients(
        @Query("query") query: String,
        @Query("type") type: String = "auto"
    ): Response<ApiResponse<RecipientSearchData>>

    @POST("wallet/transfers")
    suspend fun createTransfer(@Body body: CreateTransferRequest): Response<ApiResponse<TransferData>>

    @GET("wallet/transfers/{transferId}")
    suspend fun transfer(@Path("transferId") transferId: String): Response<ApiResponse<TransferData>>

    @GET("wallet/records")
    suspend fun walletRecords(
        @Query("limit") limit: Int = 20,
        @Query("cursor") cursor: String? = null,
        @Query("afterRecordId") afterRecordId: String? = null
    ): Response<ApiResponse<WalletRecordsData>>

    @GET("chat/messages")
    suspend fun chatMessages(
        @Query("limit") limit: Int = 100,
        @Query("before") before: String? = null
    ): Response<ApiResponse<ChatMessagesData>>

    @GET("chat/presence")
    suspend fun presence(): Response<ApiResponse<PresenceData>>

    @GET("chat/online-players")
    suspend fun onlinePlayers(): Response<ApiResponse<OnlinePlayersData>>

    @GET("chat/player-directory")
    suspend fun playerDirectory(): Response<ApiResponse<PlayerDirectoryData>>

    @GET("chat/follows")
    suspend fun follows(): Response<ApiResponse<com.deuterium.app.data.FollowedPlayersData>>

    @POST("chat/follows")
    suspend fun followPlayer(@Body body: PlayerFollowRequest): Response<ApiResponse<PlayerFollowData>>

    @DELETE("chat/follows/{playerRef}")
    suspend fun unfollowPlayer(@Path("playerRef") playerRef: String): Response<ApiResponse<PlayerFollowData>>

    @GET("ai/me")
    suspend fun aiMe(): Response<ApiResponse<AiMeData>>

    @GET("ai/plans")
    suspend fun aiPlans(): Response<ApiResponse<AiPlansData>>

    @GET("ai/messages")
    suspend fun aiMessages(
        @Query("limit") limit: Int = 50,
        @Query("before") before: String? = null
    ): Response<ApiResponse<AiMessagesData>>

    @POST("ai/conversation/reset")
    suspend fun resetAiConversation(
        @Body body: AiConversationResetRequest = AiConversationResetRequest()
    ): Response<ApiResponse<AiConversationResetData>>

    @POST("ai/purchases")
    suspend fun createAiPurchase(@Body body: AiPurchaseRequest): Response<ApiResponse<AiPurchaseData>>

    @GET("ai/purchases/{purchaseId}")
    suspend fun aiPurchase(@Path("purchaseId") purchaseId: String): Response<ApiResponse<AiPurchaseData>>
}
