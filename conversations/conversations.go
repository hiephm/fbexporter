package conversations

import (
	"fmt"
	"io/ioutil"
	"os"
	"text/template"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/hiephm/fbexporter/commands"
	"github.com/hiephm/fbexporter/config"
	"github.com/hiephm/fbexporter/util"
	fb "github.com/huandu/facebook"
	"github.com/pkg/errors"
	"github.com/urfave/cli"
)

type Message struct {
	Text        string `facebook:"message"`
	From        User   `facebook:"from"`
	CreatedTime string `facebook:"created_time"`
}

type MessageResult struct {
	Data []Message `facebook:"data"`
}

func init() {
	commands.Add(
		cli.Command{
			Name:   "messages",
			Usage:  "export all conversations on page",
			Action: exportMessages,
			Flags: []cli.Flag{
				cli.BoolFlag{
					Name:  "longlived,ll",
					Usage: "Use long lived token (for task that expect to run more than 2 hours)",
				},
				cli.StringFlag{
					Name:  "template,t",
					Usage: "Template file to generate from, required",
				},
				cli.StringFlag{
					Name:  "output,o",
					Usage: "Output file. If not specify, os.Stdout will be used instead",
				},
			},
		},
	)
}

func exportMessages(c *cli.Context) error {
	err := config.Init(c.GlobalString("config"))
	if err != nil {
		return errors.Wrap(err, "init config")
	}
	accessToken := config.FB.ShortLivedToken
	if c.BoolT("longlived") {
		if config.FB.LongLivedToken == "" {
			if config.FB.AppId == "" || config.FB.AppSecret == "" {
				return errors.New("AppId and AppSecrect is required for getting long lived token")
			}
			app := fb.App{}
			app.AppId = config.FB.AppId
			app.AppSecret = config.FB.AppSecret
			longLivedToken, expired, err := app.ExchangeToken(accessToken)
			if err != nil {
				return errors.Wrap(err, "fb.ExchangeToken")
			}
			if expired > 0 {
				log.Info("Long Lived Token Expiration: ", time.Unix(int64(expired), 0).Format("2006-01-02 03:04:05"))
			}
			config.FB.LongLivedToken = longLivedToken
			err = config.Save()
			if err != nil {
				log.Warn("Cannot save long lived token to config: ", err)
			}
		}
		accessToken = config.FB.LongLivedToken
	}
	templateFile := c.String("template")
	if templateFile == "" {
		return errors.New("Template file (--template) is required")
	}
	tmpl := template.New("messages")
	templateBytes, err := ioutil.ReadFile(templateFile)
	if err != nil {
		return errors.Wrap(err, "read template file")
	}
	_, err = tmpl.Parse(string(templateBytes))
	if err != nil {
		return errors.Wrap(err, "parse template file")
	}

	output := os.Stdout
	if outputFile := c.String("output"); outputFile != "" {
		output, err = os.OpenFile(outputFile, os.O_APPEND|os.O_RDWR|os.O_CREATE, 0664)
		if err != nil {
			return errors.Wrap(err, "open output file")
		}
	}

	session := fb.Session{}
	session.SetAccessToken(accessToken)

	res, err := session.Get(fmt.Sprintf("/%s/conversations", config.FB.PageId), fb.Params{})
	if err != nil {
		return errors.Wrap(err, "fb.GET /pageId/conversations")
	}

	// create a paging structure.
	paging, _ := res.Paging(&session)
	noMore := false
	pageNumber := 1
	for !noMore {
		log.Infof("Process conversations page %d", pageNumber)
		pageNumber++
		convResult := ConversationResult{}
		err = paging.Decode(&convResult)
		if err != nil {
			log.Warn("Decode conversations: ", err)
			noMore, _ = paging.Next()
			continue
		}

		for _, conversation := range convResult.Data {
			convLog := log.WithField("conversationId", conversation.ID)
			convSession := fb.Session{}
			convSession.SetAccessToken(accessToken)
			res, err = convSession.Get(fmt.Sprintf("/%s/messages", conversation.ID), fb.Params{"fields": "message,from,created_time"})

			if err != nil {
				convLog.Warn("fb.GET /conversationId: ", err)
			}

			msgPaging, _ := res.Paging(&session)
			noMoreMsg := false
			msgPageNumber := 1

			for !noMoreMsg {
				log.Infof("Process messages page %d", msgPageNumber)
				msgPageNumber++
				result := MessageResult{}
				err = msgPaging.Decode(&result)
				if err != nil {
					convLog.Warn("decode MessageResult: ", err)
					noMoreMsg, _ = msgPaging.Next()
					continue
				}
				if len(result.Data) > 0 {
					for i, message := range result.Data {
						message.CreatedTime, _ = util.ToSqlTime(message.CreatedTime)
						message.Text = util.EscapeString(message.Text)
						message.From.Name = util.EscapeString(message.From.Name)
						result.Data[i] = message
					}
					err = tmpl.Execute(output, map[string]interface{}{
						"conversation_id": conversation.ID,
						"messages":        result.Data,
					})
					if err != nil {
						log.Warn("render template to output: ", err)
					}
				} else {
					convLog.Warnf("No messages found.")
				}
				noMoreMsg, _ = msgPaging.Next()
			}
		}
		noMore, _ = paging.Next()
	}
	log.Info("DONE")
	return nil
}
