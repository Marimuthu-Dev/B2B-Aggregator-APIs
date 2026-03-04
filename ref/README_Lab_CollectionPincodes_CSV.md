# Lab Master – Collection Pincodes CSV Upload (FE Reference)

## Overview

Lab Master **POST** (create) and **PUT** (update) support an optional CSV file to set **CollectionPincodes** (field service / collection pincodes). If the file is sent, its pincodes are used; otherwise the value from the JSON body is used.

## CSV format

- **First row:** header with column name **`Pincodes`** (case-insensitive).
- **Data rows:** pincodes as comma-separated values. You can use either:
  - **One row:** all pincodes on a single line, comma-separated (recommended).
  - **Multiple rows:** one pincode per row (single column).
- Empty rows and empty cells are skipped.

Example (comma-separated on one row):

```csv
Pincodes
10001,10002,10003,110001,400001,560001
```

A sample file is provided: **`collection_pincodes_sample.csv`**.

## Multipart request (when using the CSV)

- **Content-Type:** `multipart/form-data`
- **Form fields:**
  - **`data`** (required): JSON string of the full lab payload (same shape as the normal POST/PUT JSON body).
  - **`file`** (optional): the CSV file. If present, **CollectionPincodes** is taken from this file and overrides any value in `data`.

## Important for frontend

- The **form field name** for the file must be **`file`**. The **filename** of the CSV (e.g. `collection_pincodes.csv`, `pincodes.csv`) does not matter; the API only checks the form field name.
- **POST** `POST /labs` – form field `data` = create body JSON, optional `file` = CSV.
- **PUT** `PUT /labs/:id` – form field `data` = update body JSON, optional `file` = CSV.

## Database

When a valid CSV is sent, the backend parses it into a comma-separated list and saves it in the **CollectionPincodes** column (e.g. `"10001,10002,10003"`). Create and update both persist this to the database correctly.
