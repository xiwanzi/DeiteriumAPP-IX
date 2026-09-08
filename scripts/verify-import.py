"""Verify the frozen import baseline; later intentional edits require a new baseline.

Text hashes normalize CRLF to LF, matching cross-platform Git checkouts.
Binary files are checked byte-for-byte. No network or private state is read.
"""
from pathlib import Path
import hashlib
import json


def digest(data):
    try:
        data.decode("utf-8")
        if b"\0" not in data:
            data = data.replace(b"\r\n", b"\n")
    except UnicodeDecodeError:
        pass
    return hashlib.sha256(data).hexdigest()


def main():
    root = Path(__file__).resolve().parents[1]
    manifest = json.loads((root / "docs/source-import-manifest.json").read_text(encoding="utf-8"))
    errors = []
    for entry in manifest["files"]:
        path = (root / entry["path"]).resolve()
        if not path.is_relative_to(root) or not path.is_file():
            errors.append(entry["path"] + ": missing or outside repository")
        elif digest(path.read_bytes()) != entry["importedSha256Lf"]:
            errors.append(entry["path"] + ": differs from import baseline")
    print(json.dumps({"checked": len(manifest["files"]), "errors": errors}, ensure_ascii=False))
    raise SystemExit(bool(errors))


if __name__ == "__main__":
    main()
