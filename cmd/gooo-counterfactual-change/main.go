package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/kimjooyoon/gooo-counterfactual-change/internal/counterfactual"
)

func main() {
	if len(os.Args) < 2 {
		fail("expected compile or evaluate")
	}
	switch os.Args[1] {
	case "compile":
		compile(os.Args[2:])
	case "evaluate":
		evaluate(os.Args[2:])
	default:
		fail("unknown command %q", os.Args[1])
	}
}

func compile(args []string) {
	flags := flag.NewFlagSet("compile", flag.ExitOnError)
	source := flags.String("source", "", "Gooo source declaration")
	contract := flags.String("contract", "", "fixed denominator contract")
	ir := flags.String("ir", "", "semantic IR output")
	flags.Parse(args)
	result, err := counterfactual.Compile(*source, *contract, *ir)
	if err != nil {
		fail("compile: %v", err)
	}
	fmt.Println(result.IRDigest)
}

func evaluate(args []string) {
	flags := flag.NewFlagSet("evaluate", flag.ExitOnError)
	options := counterfactual.EvaluateOptions{}
	flags.StringVar(&options.Source, "source", "", "Gooo source declaration")
	flags.StringVar(&options.Contract, "contract", "", "fixed denominator contract")
	flags.StringVar(&options.IR, "ir", "", "semantic IR")
	flags.StringVar(&options.Cases, "cases", "", "counterfactual fixture corpus")
	flags.StringVar(&options.BaselineFixture, "baseline-fixture", "", "immutable baseline fixture")
	flags.StringVar(&options.GeneratedGo, "generated-go", "", "generated evaluator source")
	flags.StringVar(&options.Evaluator, "evaluator", "", "independent evaluator source")
	flags.StringVar(&options.ArtifactDir, "artifact-dir", "", "empty caller-owned output directory")
	flags.StringVar(&options.ExecutionMode, "execution-mode", "", "isolated execution mode")
	flags.StringVar(&options.SubjectSHA, "subject-sha", "", "subject revision")
	flags.StringVar(&options.GoVersion, "go-version", "", "CI Go version")
	bindInt(flags, &options.Metrics.Directories, "directories")
	bindInt(flags, &options.Metrics.Files, "files")
	bindInt(flags, &options.Metrics.PhysicalLines, "physical-lines")
	bindInt(flags, &options.Metrics.GoFiles, "go-files")
	bindInt(flags, &options.Metrics.GoLines, "go-lines")
	bindInt(flags, &options.Metrics.GoooFiles, "gooo-files")
	bindInt(flags, &options.Metrics.GoooLines, "gooo-lines")
	bindInt(flags, &options.Authority.RepositoryWrites, "repository-writes")
	bindInt(flags, &options.Authority.LocalTestExecutions, "local-test-executions")
	bindInt(flags, &options.Authority.CrossProjectRequiredGates, "cross-project-required-gates")
	flags.Parse(args)
	compiled, err := counterfactual.ParseSource(options.Source)
	if err != nil {
		fail("evaluate source: %v", err)
	}
	contract, err := counterfactual.LoadContract(options.Contract)
	if err != nil {
		fail("evaluate contract: %v", err)
	}
	if err := counterfactual.ValidateDeclarations(compiled, contract); err != nil {
		fail("evaluate declarations: %v", err)
	}
	// Compile is performed by the workflow before evaluation. Reconstruct the
	// expected IR identity here without writing a second repository artifact.
	semanticIR, err := counterfactual.Compile(options.Source, options.Contract, options.IR)
	if err != nil {
		fail("evaluate compile: %v", err)
	}
	if err := counterfactual.Evaluate(options, semanticIR.IR); err != nil {
		fail("evaluate: %v", err)
	}
	fmt.Println("counterfactual evaluation complete")
}

func bindInt(flags *flag.FlagSet, target *int, name string) {
	flags.IntVar(target, name, 0, name)
}

func fail(format string, args ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", args...)
	os.Exit(1)
}
