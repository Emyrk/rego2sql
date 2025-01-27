# rego2sql

This library converts rego queries into SQL expressions to be used in `WHERE` clauses for the purposes of data filtering. 

If

Example:
```
$ go run cmd/rego2sql/main.go '"foo" == "bar"; 1 == 2'                                                                                
PGSQL:
('foo' = 'bar' AND 1 = 2)
```

# Why do this?

See blog posts like:

- https://remyasavithry.medium.com/elastic-search-data-filter-using-opa-to-implement-abac-c0896798b545
- https://jacky-jiang.medium.com/policy-based-data-filtering-solution-using-partial-evaluation-c8736bd089e0