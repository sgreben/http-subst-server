# http-subst-server

`http-subst-server` is a minimal HTTP server that serves files and renders templates with `$VARIABLES` in them.

A mash-up of [http-file-server](https://github.com/sgreben/http-file-server) and [subst](https://github.com/sgreben/subst).

## Contents

- [Contents](#contents)
- [Examples](#examples)
  - [Variables from arguments](#variables-from-arguments)
  - [Variables from the environment](#variables-from-the-environment)
  - [Variables and routes from the environment](#variables-and-routes-from-the-environment)
  - [Variables from files](#variables-from-files)
- [Get it](#get-it)
  - [Using `go get`](#using-go-get)
  - [Pre-built binary](#pre-built-binary)
- [Usage](#usage)

## Examples

### Variables from arguments

```sh
$ cat example/index.html
$GREETING $SUBJECT
```

```sh
$ http-subst-server -v GREETING=hello -v SUBJECT=world /=example
2018/12/16 09:49:09 serving "./example" on "/"
2018/12/16 09:49:09 http-subst-server listening on ":8080"
```


```sh
$ curl localhost:8080
hello world
```

### Variables from the environment

Environment variables with a name prefixed with `SUBST_VAR_` (or a custom prefix set via `-var-prefix`) can be used in the templates. The prefix is stripped from the variable name, so e.g. `$SUBST_VAR_hello` is `$hello` in the templates.


```sh
$ cat example/index.html
$GREETING $SUBJECT
```

```sh
$ export SUBST_VAR_GREETING=hello
$ export SUBST_VAR_SUBJECT=world
$ http-subst-server /=example
2018/12/16 09:49:09 serving "./example" on "/"
2018/12/16 09:49:09 http-subst-server listening on ":8080"
```

```sh
$ curl localhost:8080
hello world
```

### Variables and routes from the environment

The values of environment variables with names prefixed `SUBST_ROUTE_` (or a custom prefix set via `-route-prefix`) are used as routes. Apart from the name prefix, only the value of the environment variable is relevant.

```sh
$ cat example/index.html
$GREETING $SUBJECT
```

```sh
$ export SUBST_VAR_GREETING=hello
$ export SUBST_VAR_SUBJECT=world
$ export SUBST_ROUTE_1=/=example
$ export SUBST_ROUTE_2=/abc=/tmp
$ http-subst-server
2018/12/16 09:49:09 serving "./example" on "/"
2018/12/16 09:49:09 serving "/tmp" on "/abc/"
2018/12/16 09:49:09 http-subst-server listening on ":8080"
```

```sh
$ curl localhost:8080
hello world
```

### Variables from files

You may specify files containing variable definitions `NAME=VALUE` (one per line) using `-variable-file`/`-f`.
Changes to the definitions in these files will be picked up without having to restart the server.

```sh
$ cat example/index.html
$GREETING $SUBJECT
```

```sh
$ http-subst-server -f variables.env /=example
2018/12/16 09:49:09 serving "./example" on "/"
2018/12/16 09:49:09 http-subst-server listening on ":8080"
```

```sh
$ curl localhost:8080
$GREETING $SUBJECT
```

```sh
$ echo GREETING=hello > variables.env
$ cat variables.env
GREETING=hello
```

```sh
$ curl localhost:8080
hello $SUBJECT
```

```sh
$ echo SUBJECT=world >> variables.env
$ cat variables.env
GREETING=hello
SUBJECT=world
```

```sh
$ curl localhost:8080
hello world
```

## Get it

### Using `go get`

```sh
go get -u github.com/sgreben/http-subst-server
```

### Pre-built binary

[Download a binary](https://github.com/sgreben/http-subst-server/releases/latest) from the releases page or from the shell:

```sh
# Linux
curl -L https://github.com/sgreben/http-subst-server/releases/download/1.1.0/http-subst-server_1.1.0_linux_x86_64.tar.gz | tar xz

# OS X
curl -L https://github.com/sgreben/http-subst-server/releases/download/1.1.0/http-subst-server_1.1.0_osx_x86_64.tar.gz | tar xz

# Windows
curl -LO https://github.com/sgreben/http-subst-server/releases/download/1.1.0/http-subst-server_1.1.0_windows_x86_64.zip
unzip http-subst-server_1.1.0_windows_x86_64.zip
```

## Usage

```text
http-subst-server [OPTIONS] [[ROUTE=]PATH...]
```

```text
Usage of http-subst-server:
  -a string
    	(alias for -addr) (default ":8080")
  -addr string
    	address to listen on (environment variable "ADDR") (default ":8080")
  -escape string
    	set the escape string - a '$' preceded by this string is not treated as a variable (default "\\")
  -f value
    	(alias for -variable-file)
  -p int
    	(alias for -port)
  -port int
    	port to listen on (overrides -addr port) (environment variable "PORT")
  -q	(alias for -quiet)
  -quiet
    	disable all log output (environment variable "QUIET")
  -r value
    	(alias for -route)
  -route value
    	a route definition ROUTE=PATH (ROUTE defaults to basename of PATH if omitted)
  -route-prefix string
    	use values of environment variables with this prefix as routes (default "SUBST_ROUTE_")
  -template-suffix string
    	replace $variables in files with this suffix (default ".html")
  -undefined value
    	handling of undefined $variables, one of [ignore empty error] (default ignore)
  -v value
    	(alias for -variable)
  -var-prefix string
    	use environment variables with this prefix in templates (default "SUBST_VAR_")
  -variable value
    	a variable definition NAME[=VALUE] (if the value is omitted, the value of the environment variable with the given name is used)
  -variable-file value
    	a file consisting of lines with one variable definition NAME=VALUE per line
```
