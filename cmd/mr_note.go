package cmd

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"runtime"
	"strconv"
	"strings"
	"text/template"

	"github.com/rsteube/carapace"
	"github.com/spf13/cobra"
	gitlab "github.com/xanzy/go-gitlab"
	"github.com/zaquestion/lab/internal/action"
	"github.com/zaquestion/lab/internal/git"
	lab "github.com/zaquestion/lab/internal/gitlab"
)

var mrCreateNoteCmd = &cobra.Command{
	Use:     "note [remote] <id>",
	Aliases: []string{"comment"},
	Short:   "Add a note or comment to an MR on GitLab",
	Long:    ``,
	Args:    cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		rn, mrNum, err := parseArgs(args)
		if err != nil {
			log.Fatal(err)
		}

		msgs, err := cmd.Flags().GetStringArray("message")
		if err != nil {
			log.Fatal(err)
		}

		filename, err := cmd.Flags().GetString("file")
		if err != nil {
			log.Fatal(err)
		}

		body := ""
		if filename != "" {
			content, err := ioutil.ReadFile(filename)
			if err != nil {
				log.Fatal(err)
			}
			body = string(content)
		} else {
			body, err = mrNoteMsg(msgs)
			if err != nil {
				_, f, l, _ := runtime.Caller(0)
				log.Fatal(f+":"+strconv.Itoa(l)+" ", err)
			}
		}

		if body == "" {
			log.Fatal("aborting note due to empty note msg")
		}

		linebreak, _ := cmd.Flags().GetBool("force-linebreak")
		if linebreak {
			body = textToMarkdown(body)
		}

		noteURL, err := lab.MRCreateNote(rn, int(mrNum), &gitlab.CreateMergeRequestNoteOptions{
			Body: &body,
		})
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println(noteURL)
	},
}

func mrNoteMsg(msgs []string) (string, error) {
	if len(msgs) > 0 {
		return strings.Join(msgs[0:], "\n\n"), nil
	}

	text, err := mrNoteText()
	if err != nil {
		return "", err
	}
	return git.EditFile("MR_NOTE", text)
}

func mrNoteText() (string, error) {
	const tmpl = `{{.InitMsg}}
{{.CommentChar}} Write a message for this note. Commented lines are discarded.`

	initMsg := "\n"
	commentChar := git.CommentChar()

	t, err := template.New("tmpl").Parse(tmpl)
	if err != nil {
		return "", err
	}

	msg := &struct {
		InitMsg     string
		CommentChar string
	}{
		InitMsg:     initMsg,
		CommentChar: commentChar,
	}

	var b bytes.Buffer
	err = t.Execute(&b, msg)
	if err != nil {
		return "", err
	}

	return b.String(), nil
}

func init() {
	mrCreateNoteCmd.Flags().StringArrayP("message", "m", []string{}, "Use the given <msg>; multiple -m are concatenated as separate paragraphs")
	mrCreateNoteCmd.Flags().StringP("file", "F", "", "Use the given file as the message")
	mrCreateNoteCmd.Flags().Bool("force-linebreak", false, "append 2 spaces to the end of each line to force markdown linebreaks")

	mrCmd.AddCommand(mrCreateNoteCmd)
	carapace.Gen(mrCreateNoteCmd).PositionalCompletion(
		action.Remotes(),
		action.MergeRequests(mrList),
	)
}
