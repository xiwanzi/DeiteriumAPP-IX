"""Validate actual synthetic HTTP responses against the copied App OpenAPI."""
import argparse
import json
from pathlib import Path

import yaml
from openapi_schema_validator import OAS30Validator
from referencing import Registry, Resource
from referencing.jsonschema import DRAFT4

parser = argparse.ArgumentParser()
parser.add_argument("samples", type=Path)
parser.add_argument("--contract", type=Path)
args = parser.parse_args()
contract_path = args.contract or Path(__file__).resolve().parents[2] / "docs/contracts/openapi-app-v2.yaml"
contract_path = contract_path.resolve()
contract = yaml.safe_load(contract_path.read_text(encoding="utf-8"))
uri = contract_path.as_uri()
registry = Registry().with_resource(uri, Resource.from_contents(contract, default_specification=DRAFT4))
app_path = Path(__file__).resolve().parents[2] / "docs/contracts/openapi-app-v2.yaml"
if app_path.resolve() != contract_path:
    registry = registry.with_resource(app_path.resolve().as_uri(), Resource.from_contents(yaml.safe_load(app_path.read_text(encoding="utf-8")), default_specification=DRAFT4))
samples = json.loads(args.samples.read_text(encoding="utf-8"))
for schema, response in samples.items():
    OAS30Validator({"$ref": f"{uri}#/components/schemas/{schema}"}, registry=registry).validate(response)
print(f"Validated {len(samples)} actual HTTP responses against {contract_path.name}")
