package main

import (
	"flag"
	"fmt"
	"log"
)

const usage = `usage: daql [-dir=<path>] [-db=<path|string>] <command> [<args>]

Configuration flags:

   -dir        The project directory where the project and manifest file can be found.
               If this flag is not set, the current directory and its parents will be searched.

   -db         The database path or connection string. The environment variable DAQL_DB is used
               if this flag is not set. The string is interpreted as a dataset identifier.

Model versioning commands
   status      Check and display the model version manifest for the current project
   record      Write the current project changes to the project history and manifest

Code generation commands
   gen         Generate code for the current project
   gengo       Generate go code for specific schemas
   genpg       Generate postgres sql for specific schemas

Dataset commands
   dump        Write a specific model data stream from db to stdout
   backup      Write the db dataset to a path
   replay      Replay a dataset to the db
   migrate     Migrate a dataset and write it to a path
   scrub       Scrub a dataset and write it to a path

Other commands
   help        Display help message
   repl        Runs a read-eval-print-loop for queries to db
`

var (
	dirFlag = flag.String("dir", ".", "project directory path")
	dbFlag  = flag.String("db", "", "database connection string")
)

func main() {
	flag.Parse()
	log.SetFlags(0)
	args := flag.Args()
	if len(args) == 0 {
		log.Printf("missing command\n\n")
		fmt.Print(usage)
		return
	}
	args = args[1:]
	var err error
	switch cmd := flag.Arg(0); cmd {
	case "status":
		err = status(args)
	case "record":
		err = record(args)
	case "gen":
		err = gen(args)
	case "gengo":
		err = gengo(args)
	case "genpg":
		err = genpg(args)
	case "dump":
		err = dump(args)
	case "backup":
		err = backup(args)
	case "replay":
		err = replay(args)
	case "migrate":
		err = migrate(args)
	case "scrub":
		err = scrub(args)
	case "repl":
		err = repl(args)
	case "help":
		if len(args) > 0 {
			// TODO print command help
		}
		fmt.Print(usage)
	default:
		log.Printf("unknown command: %s\n\n", cmd)
		fmt.Print(usage)
	}
	if err != nil {
		log.Fatalf("%s error: %+v\n", flag.Arg(0), err)
	}
}
