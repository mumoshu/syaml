# syaml

Sets YAML values quickly via command-line.

```console
$ go install github.com/mumoshu/syaml

$ cat test.yaml
#foo
foo:
  #aaa
  aaa:
    #bbb
    bbb: c
    #ccc
    ccc: d
  #bar
  bar:
    #baz
    baz: 2
  #hoge
  hoge:
    fuga: 3
nested:
  array:
  - 1
  - 2
  - 3
extra:
  ### map1
  map: 1

$ cat test.yaml | syaml foo.bar.baz 123
#foo
foo:
  #aaa
  aaa:
    #bbb
    bbb: c

    #ccc
    ccc: d

  #bar
  bar:
    #baz
    baz: 123

  #hoge
  hoge:
    fuga: 3
nested:
  array:
  - 1
  - 2
  - 3
extra:
  ### map1
  map: 1
```

Inspired by [sjson](https://github.com/tidwall/sjson), [the issue 132 in go-yaml](https://github.com/go-yaml/yaml/issues/132), and based on [@niemeyer's awesome and on-going work towards go-yaml v3](https://github.com/go-yaml/yaml/pull/219#issuecomment-447415914).
