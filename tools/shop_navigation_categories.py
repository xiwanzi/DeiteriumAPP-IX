"""Plan/apply the approved two-level catalog without changing product references.

Default is read-only. After deployment approval, provide an administrator token
through DEUTERIUM_ADMIN_TOKEN and explicitly pass --apply. Uses normal catalog
authorization, versions and idempotency; never connects to the database.
"""
import argparse
import json
import os
import uuid
from urllib.request import Request, urlopen
from urllib.parse import quote, urlparse

GROUPS = {
    "家具家居": ["座椅沙发", "收纳置物", "桌台家具", "床与寝具", "厨房用品", "卫浴用品", "实验室陈设"],
    "灯具照明": ["室内灯饰", "吊灯与壁灯", "灯带与照明方块", "台灯与落地灯", "造型夜灯", "庭院石灯"],
    "绿植庭院": ["绿植园艺", "花卉植物"],
    "建筑建材": ["墙地建材"],
    "装饰摆件": ["装饰摆件", "风铃", "玩偶抱枕"],
    "其他": ["枪支"],
}


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--origin", default="http://127.0.0.1:19630")
    parser.add_argument("--snapshot", help="Read-only category JSON, for an offline plan")
    parser.add_argument("--apply", action="store_true")
    args = parser.parse_args()
    origin = args.origin.rstrip("/")
    url = urlparse(origin)
    if url.scheme != "https" and not (url.scheme == "http" and url.hostname in ("127.0.0.1", "localhost", "::1")):
        parser.error("Use HTTPS or a loopback development origin")
    if args.snapshot and args.apply:
        parser.error("An offline snapshot cannot be applied; live versions must be read")
    token = os.environ.get("DEUTERIUM_ADMIN_TOKEN", "")

    def request(path, method="GET", body=None):
        if method != "GET" and not args.apply:
            raise RuntimeError("Writes require --apply")
        data = None
        if body is not None:
            body = {**body, "clientRequestId": str(uuid.uuid5(uuid.NAMESPACE_URL, origin + path + json.dumps(body, sort_keys=True)))}
            data = json.dumps(body).encode()
        req = Request(origin + "/api/v1" + path, data=data, method=method,
                      headers={"Authorization": "Bearer " + token, "Content-Type": "application/json"})
        with urlopen(req, timeout=30) as response:
            return json.load(response)

    def categories(store):
        values, cursor, seen = [], "", set()
        while True:
            result = request(f"/merchant/stores/{quote(store)}/categories?limit=100" + ("&cursor=" + quote(cursor) if cursor else ""))
            values.extend(result["data"])
            cursor = result.get("page", {}).get("nextCursor")
            if not cursor:
                return values
            if cursor in seen:
                raise RuntimeError("Repeated category cursor")
            seen.add(cursor)

    if args.snapshot:
        with open(args.snapshot, encoding="utf-8") as file:
            rows = json.load(file)
        stores = sorted({row["storeId"] for row in rows})
        by_store = {store: [row for row in rows if row["storeId"] == store] for store in stores}
    else:
        if not token:
            parser.error("Set DEUTERIUM_ADMIN_TOKEN; it is never printed or saved")
        stores = request("/merchant/me")["data"]["storeIds"]
        by_store = {store: categories(store) for store in stores}

    plan = []
    for store, rows in by_store.items():
        for order, (group, names) in enumerate(GROUPS.items()):
            plan.append({"storeId": store, "group": group, "sortOrder": order,
                         "children": [{"categoryId": row.get("categoryId", row.get("id")), "name": row["name"]}
                                      for row in rows if row["name"] in names and row["name"] != group]})
    print(json.dumps({"mode": "apply" if args.apply else "read-only plan", "groups": plan}, ensure_ascii=False, indent=2))
    if not args.apply:
        return
    for store, rows in by_store.items():
        for order, (group, names) in enumerate(GROUPS.items()):
            candidates = [row for row in rows if row["name"] == group and not row.get("parentCategoryId") and row.get("active")]
            if len(candidates) > 1:
                raise RuntimeError(f"Ambiguous root category: {store} / {group}")
            parent = candidates[0] if candidates else request(f"/merchant/stores/{quote(store)}/categories", "POST",
                {"name": group, "sortOrder": order, "active": True, "parentCategoryId": None})["data"]
            if parent["sortOrder"] != order:
                parent = request(f"/merchant/stores/{quote(store)}/categories/{quote(parent['categoryId'])}", "PUT",
                    {"expectedVersion": parent["version"], "name": group, "sortOrder": order, "active": True, "parentCategoryId": None})["data"]
            for row in rows:
                if row["name"] not in names or row["categoryId"] == parent["categoryId"] or row.get("parentCategoryId") == parent["categoryId"]:
                    continue
                if row.get("parentCategoryId"):
                    raise RuntimeError(f"Existing manual hierarchy requires review: {row['name']}")
                request(f"/merchant/stores/{quote(store)}/categories/{quote(row['categoryId'])}", "PUT",
                    {"expectedVersion": row["version"], "name": row["name"], "sortOrder": row["sortOrder"],
                     "active": row["active"], "parentCategoryId": parent["categoryId"]})
    print("Approved grouping applied; product IDs, references and versions unchanged.")


if __name__ == "__main__":
    main()
