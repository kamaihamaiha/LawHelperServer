#!/usr/bin/env python3
"""将 law_json 目录下解析文件的 publish_date 与数据库 laws_list.publishDate 同步"""

import sqlite3
import json
import os
import glob
import sys

DB_PATH = os.path.join(os.path.dirname(__file__), "..", "data", "db", "laws.db")
JSON_DIR = os.path.join(os.path.dirname(__file__), "..", "data", "law_json", "laws_by_type")


def main():
    db = sqlite3.connect(DB_PATH)
    rows = db.execute(
        "SELECT versionId, publishDate FROM laws_list WHERE publishDate IS NOT NULL AND publishDate != ''"
    ).fetchall()
    db_map = {r[0]: r[1] for r in rows}
    db.close()

    updated = 0
    skipped = 0

    for fpath in glob.glob(os.path.join(JSON_DIR, "*", "*.json")):
        with open(fpath, "r", encoding="utf-8") as f:
            data = json.load(f)

        vid = data.get("law_id", "")
        if vid not in db_map:
            skipped += 1
            continue

        if data.get("publish_date") == db_map[vid]:
            continue

        data["publish_date"] = db_map[vid]

        with open(fpath, "w", encoding="utf-8") as f:
            json.dump(data, f, ensure_ascii=False, indent=4)
            f.write("\n")

        updated += 1

    print(f"更新: {updated} 个文件")
    print(f"跳过: {skipped} 个文件 (数据库中无记录)")


if __name__ == "__main__":
    main()
