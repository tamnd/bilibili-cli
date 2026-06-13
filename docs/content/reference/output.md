---
title: "Output formats"
description: "Every output format, how to narrow columns, and how to template rows."
weight: 30
---

Every list command renders through the same formatter. Pick a format with `-o`,
or let bili choose: a table when writing to a terminal, JSONL when piped.

## Formats

```bash
bili search lofi -o table   # aligned columns for reading
bili search lofi -o jsonl   # one JSON object per line, for piping
bili search lofi -o json    # a single JSON array
bili search lofi -o csv     # spreadsheet friendly
bili search lofi -o tsv     # tab-separated
bili search lofi -o yaml    # YAML documents
bili search lofi -o url     # just the URL column
bili search lofi -o raw     # the underlying record as pretty-printed JSON
```

| Format | Best for |
|---|---|
| `table` | Reading on a terminal |
| `jsonl` | Piping into another tool, one object at a time |
| `json` | Loading a whole result as an array |
| `csv` / `tsv` | Spreadsheets and quick column math |
| `yaml` | Reading a single rich record top to bottom |
| `url` | Feeding URLs into other commands |
| `raw` | The full record, pretty-printed |

## Rich, lossless records

bili keeps the fields the API returns rather than flattening them. The `table`,
`csv`, and `url` views are readable projections; `json`, `jsonl`, and `yaml` are
the complete record. When in doubt about what a command knows, ask for JSON:

```bash
bili video BV17x411w7KC -o json | jq 'keys'
```

## Narrowing columns

Keep only the fields you want, in the order you list them:

```bash
bili search lofi --fields bvid,title,view_count
```

`--no-header` drops the header row in `table` and `csv` output, which is handy
when a downstream tool expects bare rows.

## Templating rows

For full control over each line, apply a Go `text/template`. Fields are the JSON
keys, capitalised:

```bash
bili search lofi --template '{{.BVID}} {{.Title}}'
bili video BVID --template '{{.Title}} — {{.ViewCount}} views'
```

## Why auto-detection helps

Because the default adapts to the destination, the same command reads well by
hand and parses cleanly in a pipe:

```bash
bili search lofi            # a table, because this is a terminal
bili search lofi | wc -l    # JSONL, because this is a pipe
```

You only reach for `-o` when you want something other than that default.
