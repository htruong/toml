Usage
------

```go
var doc=`
	age = 17
`

type Config struct {
	Age int
}

var config Config
toml.Unmarshal(doc, &config)
```

Generic Decode

```go
var config interface{}  // or map[string]interface{}
toml.Unmarshal(doc, &config)
```
