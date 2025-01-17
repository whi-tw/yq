package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/mikefarah/yq/v4/pkg/yqlib"
	"github.com/spf13/cobra"
)

func createEvaluateSequenceCommand() *cobra.Command {
	var cmdEvalSequence = &cobra.Command{
		Use:     "eval [expression] [yaml_file1]...",
		Aliases: []string{"e"},
		Short:   "(default) Apply the expression to each document in each yaml file in sequence",
		Example: `
# Reads field under the given path for each file
yq e '.a.b' f1.yml f2.yml 

# Prints out the file
yq e sample.yaml 

# Pipe from STDIN
## use '-' as a filename to pipe from STDIN
cat file2.yml | yq e '.a.b' file1.yml - file3.yml

# Creates a new yaml document
## Note that editing an empty file does not work.
yq e -n '.a.b.c = "cat"' 

# Update a file inplace
yq e '.a.b = "cool"' -i file.yaml 
`,
		Long: `yq is a portable command-line YAML processor (https://github.com/mikefarah/yq/) 
See https://mikefarah.gitbook.io/yq/ for detailed documentation and examples.

## Evaluate Sequence ##
This command iterates over each yaml document from each given file, applies the 
expression and prints the result in sequence.`,
		RunE: evaluateSequence,
	}
	return cmdEvalSequence
}

func processExpression(expression string) string {

	if prettyPrint && expression == "" {
		return yqlib.PrettyPrintExp
	} else if prettyPrint {
		return fmt.Sprintf("%v | %v", expression, yqlib.PrettyPrintExp)
	}
	return expression
}

func evaluateSequence(cmd *cobra.Command, args []string) (cmdError error) {
	// 0 args, read std in
	// 1 arg, null input, process expression
	// 1 arg, read file in sequence
	// 2+ args, [0] = expression, file the rest

	var err error
	firstFileIndex, err := initCommand(cmd, args)
	if err != nil {
		return err
	}

	stat, _ := os.Stdin.Stat()
	pipingStdIn := (stat.Mode() & os.ModeCharDevice) == 0
	yqlib.GetLogger().Debug("pipingStdIn: %v", pipingStdIn)

	yqlib.GetLogger().Debug("stat.Mode(): %v", stat.Mode())
	yqlib.GetLogger().Debug("ModeDir: %v", stat.Mode()&os.ModeDir)
	yqlib.GetLogger().Debug("ModeAppend: %v", stat.Mode()&os.ModeAppend)
	yqlib.GetLogger().Debug("ModeExclusive: %v", stat.Mode()&os.ModeExclusive)
	yqlib.GetLogger().Debug("ModeTemporary: %v", stat.Mode()&os.ModeTemporary)
	yqlib.GetLogger().Debug("ModeSymlink: %v", stat.Mode()&os.ModeSymlink)
	yqlib.GetLogger().Debug("ModeDevice: %v", stat.Mode()&os.ModeDevice)
	yqlib.GetLogger().Debug("ModeNamedPipe: %v", stat.Mode()&os.ModeNamedPipe)
	yqlib.GetLogger().Debug("ModeSocket: %v", stat.Mode()&os.ModeSocket)
	yqlib.GetLogger().Debug("ModeSetuid: %v", stat.Mode()&os.ModeSetuid)
	yqlib.GetLogger().Debug("ModeSetgid: %v", stat.Mode()&os.ModeSetgid)
	yqlib.GetLogger().Debug("ModeCharDevice: %v", stat.Mode()&os.ModeCharDevice)
	yqlib.GetLogger().Debug("ModeSticky: %v", stat.Mode()&os.ModeSticky)
	yqlib.GetLogger().Debug("ModeIrregular: %v", stat.Mode()&os.ModeIrregular)

	// Mask for the type bits. For regular files, none will be set.
	yqlib.GetLogger().Debug("ModeType: %v", stat.Mode()&os.ModeType)

	yqlib.GetLogger().Debug("ModePerm: %v", stat.Mode()&os.ModePerm)

	out := cmd.OutOrStdout()

	if writeInplace {
		// only use colors if its forced
		colorsEnabled = forceColor
		writeInPlaceHandler := yqlib.NewWriteInPlaceHandler(args[firstFileIndex])
		out, err = writeInPlaceHandler.CreateTempFile()
		if err != nil {
			return err
		}
		// need to indirectly call the function so  that completedSuccessfully is
		// passed when we finish execution as opposed to now
		defer func() {
			if cmdError == nil {
				cmdError = writeInPlaceHandler.FinishWriteInPlace(completedSuccessfully)
			}
		}()
	}

	format, err := yqlib.OutputFormatFromString(outputFormat)
	if err != nil {
		return err
	}

	printerWriter, err := configurePrinterWriter(format, out)
	if err != nil {
		return err
	}
	encoder := configureEncoder(format)

	printer := yqlib.NewPrinter(encoder, printerWriter)

	decoder, err := configureDecoder()
	if err != nil {
		return err
	}
	streamEvaluator := yqlib.NewStreamEvaluator()

	if frontMatter != "" {
		yqlib.GetLogger().Debug("using front matter handler")
		frontMatterHandler := yqlib.NewFrontMatterHandler(args[firstFileIndex])
		err = frontMatterHandler.Split()
		if err != nil {
			return err
		}
		args[firstFileIndex] = frontMatterHandler.GetYamlFrontMatterFilename()

		if frontMatter == "process" {
			reader := frontMatterHandler.GetContentReader()
			printer.SetAppendix(reader)
			defer yqlib.SafelyCloseReader(reader)
		}
		defer frontMatterHandler.CleanUp()
	}
	expression, args := processArgs(pipingStdIn, args)

	switch len(args) {
	case 0:
		if nullInput {
			err = streamEvaluator.EvaluateNew(processExpression(expression), printer, "")
		} else {
			cmd.Println(cmd.UsageString())
			return nil
		}
	default:
		err = streamEvaluator.EvaluateFiles(processExpression(expression), args, printer, leadingContentPreProcessing, decoder)
	}
	completedSuccessfully = err == nil

	if err == nil && exitStatus && !printer.PrintedAnything() {
		return errors.New("no matches found")
	}

	return err
}
