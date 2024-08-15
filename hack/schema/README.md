# schema generation

This go project is designed to generate the JSON schema for the given API version of zarf.yaml files.

## Usage
Run the program with the desired API version as an argument:
```bash
go run main.go v1alpha1
```
The generated JSON schema will be printed to the console.

Alternatively run `./create-zarf-schema.sh` which will generate all of the schemas, add yaml extension, and move the schema files to their proper place in the repo.
