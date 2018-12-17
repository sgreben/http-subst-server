package main

import (
	"bufio"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	defaultAddr          = ":8080"
	addrEnvVarName       = "ADDR"
	portEnvVarName       = "PORT"
	quietEnvVarName      = "QUIET"
	rootRoute            = "/"
	undefinedValueEmpty  = "empty"
	undefinedValueIgnore = "ignore"
	undefinedValueError  = "error"
)

var (
	substEnvVarNamePrefix  = "SUBST_VAR_"
	routeEnvVarNamePrefix  = "SUBST_ROUTE_"
	templateFileNameSuffix = ".html"
	addrFlag               = os.Getenv(addrEnvVarName)
	portFlag64, _          = strconv.ParseInt(os.Getenv(portEnvVarName), 10, 64)
	portFlag               = int(portFlag64)
	quietFlag              bool
	routesFlag             routes
	substVariables         variables
	variableFilesFlag      stringsVar
	variablesFromStdinFlag bool
	reloadInterval         time.Duration
	undefinedWarn          = true
	escape                 string
	undefinedKey           = enumVar{
		Choices: []string{
			undefinedValueEmpty,
			undefinedValueIgnore,
			undefinedValueError,
		},
		Value: undefinedValueIgnore,
	}
)

func init() {
	log.SetFlags(log.LUTC | log.Ldate | log.Ltime)
	log.SetOutput(os.Stderr)
	if addrFlag == "" {
		addrFlag = defaultAddr
	}
	flag.StringVar(&addrFlag, "addr", addrFlag, fmt.Sprintf("address to listen on (environment variable %q)", addrEnvVarName))
	flag.StringVar(&addrFlag, "a", addrFlag, "(alias for -addr)")
	flag.IntVar(&portFlag, "port", portFlag, fmt.Sprintf("port to listen on (overrides -addr port) (environment variable %q)", portEnvVarName))
	flag.IntVar(&portFlag, "p", portFlag, "(alias for -port)")
	flag.BoolVar(&quietFlag, "quiet", quietFlag, fmt.Sprintf("disable all log output (environment variable %q)", quietEnvVarName))
	flag.BoolVar(&quietFlag, "q", quietFlag, "(alias for -quiet)")
	flag.Var(&routesFlag, "route", routesFlag.help())
	flag.Var(&routesFlag, "r", "(alias for -route)")
	flag.StringVar(&substEnvVarNamePrefix, "var-prefix", substEnvVarNamePrefix, "use environment variables with this prefix in templates")
	flag.StringVar(&routeEnvVarNamePrefix, "route-prefix", routeEnvVarNamePrefix, "use values of environment variables with this prefix as routes")
	flag.StringVar(&templateFileNameSuffix, "template-suffix", templateFileNameSuffix, "replace $variables in files with this suffix")
	flag.Var(&substVariables, "variable", substVariables.help())
	flag.Var(&substVariables, "v", "(alias for -variable)")
	flag.Var(&variableFilesFlag, "variable-file", "a file consisting of lines with one variable definition NAME[=VALUE] per line")
	flag.Var(&variableFilesFlag, "f", "(alias for -variable-file)")
	flag.BoolVar(&variablesFromStdinFlag, "variables-from-stdin", false, "read lines with variable definitions NAME[=VALUE] from stdin")
	flag.BoolVar(&variablesFromStdinFlag, "i", false, "(alias for -variables-from-stdin)")
	flag.DurationVar(&reloadInterval, "variable-file-reload", time.Second, "reload interval for variable files")
	flag.Var(&undefinedKey, "undefined", "handling of undefined $variables, one of [ignore empty error] (default ignore)")
	flag.StringVar(&escape, "escape", "\\", "set the escape string - a '$' preceded by this string is not treated as a variable")
	for _, e := range os.Environ() {
		if strings.HasPrefix(e, substEnvVarNamePrefix) {
			kv := strings.TrimPrefix(e, substEnvVarNamePrefix)
			if err := substVariables.Set(kv); err != nil {
				log.Printf("parse %q: %v", kv, err)
			}
		}
		if strings.HasPrefix(e, routeEnvVarNamePrefix) {
			i := strings.IndexRune(e, '=')
			kv := e[i+1:]
			if err := routesFlag.Set(kv); err != nil {
				log.Printf("parse %q: %v", kv, err)
			}
		}
	}
	flag.Parse()

	if quietFlag {
		log.SetOutput(ioutil.Discard)
	}
	for i := 0; i < flag.NArg(); i++ {
		arg := flag.Arg(i)
		err := routesFlag.Set(arg)
		if err != nil {
			log.Fatalf("%q: %v", arg, err)
		}
	}

	if len(routesFlag.Values) == 0 {
		log.Println("no routes defined, creating temporary directory to serve from")
		tempDir, err := ioutil.TempDir("", "")
		if err != nil {
			log.Fatalf("create temporary directory: %v", err)
		}
		tempDirIndexPath := filepath.Join(tempDir, indexBaseName)
		if err := ioutil.WriteFile(tempDirIndexPath, nil, 0600); err != nil {
			log.Fatalf("write empty index file %q: %v", tempDirIndexPath, err)
		}
		routesFlag.Set(fmt.Sprintf("/=%s", tempDir))
		log.Printf("created temporary directory %q. printing its path to stdout.", filepath.Base(tempDir))
		fmt.Println(tempDir)
	}

}

func main() {
	addr, err := addr()
	if err != nil {
		log.Fatalf("address/port: %v", err)
	}
	go loadVariables(reloadInterval)
	err = server(addr, routesFlag)
	if err != nil {
		log.Fatalf("start server: %v", err)
	}
}

func loadVariables(interval time.Duration) {
	tick := time.Tick(interval)
	stdinCh := make(chan string)
	if variablesFromStdinFlag {
		log.Printf("reading variable definitions NAME[=VALUE] from stdin")
		go stdinLines(stdinCh)
	}
	for {
		select {
		case line := <-stdinCh:
			if err := substVariables.Set(line); err != nil {
				log.Printf("parse %q from stdin: %v", line, err)
			}
		case <-tick:
			loadVariableFiles()
		}
	}
}

func stdinLines(out chan string) {
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		out <- scanner.Text()
	}
	if err := scanner.Err(); err != nil {
		log.Printf("scan stdin: %v", err)
	}
	log.Println("stdin closed")
}

func loadVariableFiles() {
	for _, path := range variableFilesFlag.Values {
		if err := loadVariableFile(path); err != nil {
			log.Printf("error: %v", err)
		}
	}
}

func loadVariableFile(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if err := substVariables.Set(line); err != nil {
			log.Printf("parse %q in %q: %v", line, path, err)
		}
	}
	if err := scanner.Err(); err != nil {
		log.Printf("scan %q: %v", path, err)
	}
	return nil
}

func subst(k string) (out string, err error) {
	if v, ok := substVariables.Values[k]; ok {
		return v, nil
	}
	switch undefinedKey.Value {
	case undefinedValueEmpty:
		out = ""
	case undefinedValueError:
		err = fmt.Errorf(`undefined key: $%s`, k)
	case undefinedValueIgnore:
		fallthrough
	default:
		out = "$" + k
	}
	if undefinedWarn {
		log.Printf(`undefined key: $%s, using value "%s"`, k, out)
	}
	return out, err
}

func render(template string) (string, error) {
	substVariables.Mu.RLock()
	defer substVariables.Mu.RUnlock()
	return expand(string(template), escape, subst)
}

func server(addr string, routes routes) error {
	mux := http.DefaultServeMux
	handlers := make(map[string]http.Handler)
	paths := make(map[string]string)

	for _, route := range routes.Values {
		handlers[route.Route] = &fileHandler{
			route:  route.Route,
			path:   route.Path,
			render: render,
		}
		paths[route.Route] = route.Path
	}

	for route, path := range paths {
		mux.Handle(route, handlers[route])
		log.Printf("serving %q on %q", path, route)
	}

	_, rootRouteTaken := handlers[rootRoute]
	if !rootRouteTaken {
		route := routes.Values[0].Route
		mux.Handle(rootRoute, http.RedirectHandler(route, http.StatusTemporaryRedirect))
		log.Printf("redirecting to %q from %q", route, rootRoute)
	}

	binaryPath, _ := os.Executable()
	if binaryPath == "" {
		binaryPath = "server"
	}
	log.Printf("%s listening on %q", filepath.Base(binaryPath), addr)
	return http.ListenAndServe(addr, mux)
}

func addr() (string, error) {
	portSet := portFlag != 0
	addrSet := addrFlag != ""
	switch {
	case portSet && addrSet:
		a, err := net.ResolveTCPAddr("tcp", addrFlag)
		if err != nil {
			return "", err
		}
		a.Port = portFlag
		return a.String(), nil
	case !portSet && addrSet:
		a, err := net.ResolveTCPAddr("tcp", addrFlag)
		if err != nil {
			return "", err
		}
		return a.String(), nil
	case portSet && !addrSet:
		return fmt.Sprintf(":%d", portFlag), nil
	case !portSet && !addrSet:
		fallthrough
	default:
		return defaultAddr, nil
	}
}
