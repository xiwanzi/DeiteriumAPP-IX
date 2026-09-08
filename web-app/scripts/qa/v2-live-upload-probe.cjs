async (page) => {
  await page.addScriptTag({ path: "C:/DeuteriumAPP/.worktrees/web-player-v1/web-app/node_modules/spark-md5/spark-md5.min.js" });
  return await page.evaluate(async () => {
    const sessionResponse = await fetch("/api/v1/web/session", { credentials: "same-origin", cache: "no-store" });
    if (!sessionResponse.ok) throw new Error("Upload probe requires the approved live session.");
    const session = (await sessionResponse.json()).data, csrf = session.csrfToken;
    const call = async (path, body) => {
      const response = await fetch("/api/v1" + path, { method: "POST", credentials: "same-origin", headers: { "Content-Type": "application/json", "X-CSRF-Token": csrf }, body: JSON.stringify(body) });
      const value = await response.json(); if (!response.ok) throw new Error(value.error?.code || "API_FAILURE"); return value.data;
    };
    const canvas = document.createElement("canvas"); canvas.width = 4; canvas.height = 4;
    const context = canvas.getContext("2d"); context.fillStyle = "#4383d2"; context.fillRect(0, 0, 4, 4);
    const blob = await new Promise((resolve) => canvas.toBlob(resolve, "image/png")), buffer = await blob.arrayBuffer();
    const hex = window.SparkMD5.ArrayBuffer.hash(buffer), md5 = btoa(hex.match(/../g).map((byte) => String.fromCharCode(parseInt(byte, 16))).join(""));
    let upload, result, putStatus, removed = false, failure;
    try {
      upload = await call("/assets/uploads", { clientRequestId: crypto.randomUUID(), purpose: "AVATAR", businessType: "PROFILE", businessRef: session.user.userId, fileName: "browser-connectivity-probe.png", contentType: "image/png", sizeBytes: blob.size, contentMd5: md5, altText: "Browser storage connectivity probe" });
      const auth = upload.authorization;
      if (auth.method !== "PUT" || auth.provider !== "RAINYUN_S3") throw new Error("INVALID_UPLOAD_AUTHORIZATION");
      const put = await fetch(auth.uploadUrl, { method: "PUT", credentials: "omit", headers: auth.signedHeaders, body: blob });
      putStatus = put.status; if (!put.ok) throw new Error("S3_PUT_" + put.status);
      result = await call(`/assets/uploads/${upload.uploadId}/complete`, { clientRequestId: crypto.randomUUID(), ossRequestId: "" });
      if (result.status !== "READY" || !result.asset?.assetId) throw new Error("VERIFICATION_" + result.status);
    } catch (error) { failure = error.message; }
    finally { if (upload?.assetId) { await call(`/assets/${upload.assetId}/remove`, { clientRequestId: crypto.randomUUID() }); removed = true; } }
    return { mode: "approved live unbound self-upload probe; no avatar or public content changed", browserPutStatus: putStatus, verified: result?.status === "READY", removed, failure };
  });
}
