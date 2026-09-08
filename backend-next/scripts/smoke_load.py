"""Small loopback-only baseline; never load test a production URL."""
import argparse
import concurrent.futures
import ctypes
import http.client
import json
import statistics
import time
from pathlib import Path
from urllib.parse import urlsplit

parser = argparse.ArgumentParser()
parser.add_argument("--url", default="http://127.0.0.1:18088")
parser.add_argument("--pid", type=int, required=True)
parser.add_argument("--output", type=Path, required=True)
args = parser.parse_args()
url = urlsplit(args.url)
if url.scheme != "http" or url.hostname != "127.0.0.1":
    raise SystemExit("Only a local loopback development service is allowed")


def memory():
    class Counters(ctypes.Structure):
        _fields_ = [("cb", ctypes.c_ulong), ("PageFaultCount", ctypes.c_ulong)] + [
            (name, ctypes.c_size_t)
            for name in ("PeakWorkingSetSize", "WorkingSetSize", "QuotaPeakPagedPoolUsage",
                         "QuotaPagedPoolUsage", "QuotaPeakNonPagedPoolUsage",
                         "QuotaNonPagedPoolUsage", "PagefileUsage", "PeakPagefileUsage")
        ]
    kernel = ctypes.windll.kernel32
    kernel.OpenProcess.restype = ctypes.c_void_p
    kernel.OpenProcess.argtypes = [ctypes.c_ulong, ctypes.c_int, ctypes.c_ulong]
    kernel.CloseHandle.argtypes = [ctypes.c_void_p]
    psapi = ctypes.windll.psapi
    psapi.GetProcessMemoryInfo.argtypes = [ctypes.c_void_p, ctypes.POINTER(Counters), ctypes.c_ulong]
    handle = kernel.OpenProcess(0x0400 | 0x0010, 0, args.pid)
    if not handle:
        raise OSError("Cannot inspect local backend process")
    try:
        counters = Counters()
        counters.cb = ctypes.sizeof(counters)
        if not psapi.GetProcessMemoryInfo(handle, ctypes.byref(counters), counters.cb):
            raise OSError("Cannot read process memory")
        return {"workingSetMiB": round(counters.WorkingSetSize / 2**20, 2),
                "peakWorkingSetMiB": round(counters.PeakWorkingSetSize / 2**20, 2)}
    finally:
        kernel.CloseHandle(handle)


idle = memory()
connection = http.client.HTTPConnection(url.hostname, url.port, timeout=10)
started = time.perf_counter()
connection.request("POST", "/api/v1/account/login", json.dumps({"account": "Alice", "password": "migration-password-123"}), {"Content-Type": "application/json"})
response = connection.getresponse()
login = json.loads(response.read())
if response.status != 200:
    raise RuntimeError("Synthetic fixture login failed")
login_ms = (time.perf_counter() - started) * 1000
token = login["data"]["token"]
after_login = memory()


def worker(count):
    client = http.client.HTTPConnection(url.hostname, url.port, timeout=10)
    timings = []
    try:
        for _ in range(count):
            started = time.perf_counter()
            client.request("GET", "/api/v1/account/me", headers={"Authorization": f"Bearer {token}"})
            response = client.getresponse()
            payload = json.loads(response.read())
            timings.append((time.perf_counter() - started) * 1000)
            if response.status != 200 or payload["data"]["user"]["userId"] != "legacy_alice":
                raise RuntimeError("Authenticated request failed")
    finally:
        client.close()
    return timings


started = time.perf_counter()
with concurrent.futures.ThreadPoolExecutor(max_workers=16) as pool:
    samples = [sample for batch in pool.map(worker, [64] * 16) for sample in batch]
elapsed = time.perf_counter() - started
after_load = memory()
connection.request("POST", "/api/v1/account/logout", "{}", {"Authorization": f"Bearer {token}", "Content-Type": "application/json"})
logout = connection.getresponse()
logout.read()
assert logout.status == 200
connection.close()
samples.sort()
report = {
    "platform": "Windows amd64, Go 1.27.1, MariaDB 11.8.8 on loopback",
    "dataset": "one synthetic migrated account; keep-alive HTTP; no TLS",
    "requests": len(samples), "concurrency": 16, "errors": 0,
    "elapsedSeconds": round(elapsed, 3), "requestsPerSecond": round(len(samples) / elapsed, 1),
    "meanMs": round(statistics.mean(samples), 2),
    "p50Ms": round(samples[int(len(samples) * .50) - 1], 2),
    "p95Ms": round(samples[int(len(samples) * .95) - 1], 2),
    "p99Ms": round(samples[int(len(samples) * .99) - 1], 2),
    "firstLegacyLoginIncludingHashUpgradeMs": round(login_ms, 2),
    "idle": idle, "afterLogin": after_login, "afterLoad": after_load,
    "limitations": "Not a Linux runtime test or a comparison against the old backend; excludes MySQL memory, TLS, WAN and other business workloads."
}
args.output.parent.mkdir(parents=True, exist_ok=True)
args.output.write_text(json.dumps(report, ensure_ascii=False, indent=2), encoding="utf-8")
print(json.dumps(report, ensure_ascii=False))
