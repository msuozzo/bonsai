module github.com/msuozzo/bonsai/bonsai-markdown

go 1.25.0

require (
	github.com/msuozzo/bonsai v0.0.0
	github.com/msuozzo/bonsai/bonsai-markdown-inline v0.0.0
)

replace (
	github.com/msuozzo/bonsai => ../
	github.com/msuozzo/bonsai/bonsai-markdown-inline => ../bonsai-markdown-inline
)
