test:
	go run cmd/main.go test.yaml foo.bar.baz 123

# Run this with DEBUG=1 like `DEBUG=1 make smoke` to show verbose logs for debugging
smoke:
	bash -c 'go build -o syaml ./cmd && ! diff -u <(cat test.yaml) <(cat test.yaml | ./syaml foo.bar.baz 123)'
	# The above hould print only the diff below:
	#
	# -    baz: 2
    # +    baz: 123

smoke2:
	bash -c 'go build -o syaml ./cmd && ! diff -u <(cat test.2.yaml) <(cat test.2.yaml | ./syaml foo.bar.baz 123)'
	# The above hould print only the diff below:
	#
	# -    baz: 2
    # +    baz: 123
