package cmd

// micro types
const (
	Static    string = "static (vanilla, react, vue)"
	Fullstack        = "fullstack (next, nuxt, svelte-kit)"
	Native           = "native (python, nodejs)"
	Custom           = "custom (go, rust)"
)

// frameworks
const (
	Vanilla   = "vanilla"
	React     = "react"
	Svelte    = "svelte"
	Vue       = "vue"
	Next      = "next"
	Nuxt      = "nuxt"
	SvelteKit = "svelte-kit"
	Python38  = "python3.8"
	Python39  = "python3.9"
	Node14x   = "nodejs14.x"
	Node16x   = "nodejs16.x"
	Go        = "go"
	Rust      = "rust"
)

var (
	// MicroTypes supported micro types
	MicroTypes = []string{Static, Fullstack, Native, Custom}
	// MicroTypesToFrameworks supported frameworks for each micro type
	MicroTypesToFrameworks = map[string][]string{
		Static:    {Vanilla, React, Vue, Svelte},
		Fullstack: {Next, SvelteKit, Nuxt},
		Native:    {Python38, Python39, Node14x, Node16x},
		Custom:    {Go, Rust},
	}
)
