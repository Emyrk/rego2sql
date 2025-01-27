package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/Emyrk/rego2sql"
	"github.com/open-policy-agent/opa/v1/ast"
)

func main() {
	log.SetOutput(os.Stderr)
	flag.Parse()

	bodies := make([]ast.Body, 0)
	for _, arg := range flag.Args() {
		body, err := ast.ParseBody(arg)
		if err != nil {
			log.Fatal(fmt.Errorf("parse body %s: %w", arg, err).Error())
		}
		bodies = append(bodies, body)
	}

	sqlNode, err := rego2sql.Convert(rego2sql.ConvertConfig{}, bodies)
	if err != nil {
		log.Fatal(fmt.Errorf("convert: %w", err).Error())
	}

	output, err := rego2sql.Serialize(sqlNode)
	if err != nil {
		log.Fatal(fmt.Errorf("serialize: %w", err).Error())
	}

	fmt.Println("PGSQL:")
	fmt.Println(output)
}
