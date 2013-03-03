Usage
------

	var doc=`
		age = 17
	`

	type Config struct {
		Age int
	}

	var config Config
	toml.Unmarshal(doc, &config)
