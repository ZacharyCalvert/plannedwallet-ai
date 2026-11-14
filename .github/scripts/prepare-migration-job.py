#!/usr/bin/env python3
import base64
import json
import sys


def main():
    if len(sys.argv) < 4:
        print("Usage: prepare-migration-job.py <template_json_path> <sql_file_path> <image>")
        sys.exit(1)

    template_path = sys.argv[1]
    sql_file_path = sys.argv[2]
    image = sys.argv[3]

    with open(sql_file_path, "rb") as f:
        sql_b64 = base64.b64encode(f.read()).decode("utf-8")

    with open(template_path, "r") as f:
        template = json.load(f)

    container = template["containers"][0]
    container["image"] = image
    container["command"] = ["/bin/sh"]
    container["args"] = [
        "-c",
        f"echo {sql_b64} | base64 -d | psql -v ON_ERROR_STOP=1 -h $DB_HOST -U $DB_USER -d $DB_NAME",
    ]

    with open(template_path, "w") as f:
        json.dump(template, f, indent=2)

    print(f"Successfully prepared migration template for {container.get('name', 'container')}.")


if __name__ == "__main__":
    main()
