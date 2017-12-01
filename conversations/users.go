package conversations

import (
	"fmt"
	"html/template"
	"io/ioutil"
	"os"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/hiephm/fbexporter/commands"
	"github.com/hiephm/fbexporter/config"
	"github.com/hiephm/fbexporter/util"
	fb "github.com/huandu/facebook"
	"github.com/pkg/errors"
	"github.com/urfave/cli"
)

type User struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Gender   string
	LastSend string
}

type ParticipantsResult struct {
	Participants map[string][]User `json:"participants"`
}

type Conversation struct {
	ID          string `json:"id"`
	UpdatedTime string `json:"updated_time"`
}

type ConversationResult struct {
	Data []Conversation `json:"data"`
}

func init() {
	commands.Add(
		cli.Command{
			Name:   "users",
			Usage:  "export all users that have chat with a FB page",
			Action: exportUser,
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

func exportUser(c *cli.Context) error {
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
	tmpl := template.New("users")
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
		output, err = os.OpenFile(outputFile, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0664)
		if err != nil {
			return errors.Wrap(err, "open output file")
		}
	}

	session := fb.Session{}
	session.SetAccessToken(accessToken)

	res, err := session.Get("/me", fb.Params{"fields": "id,name"})
	if err != nil {
		return errors.Wrap(err, "fb.GET /me")
	}

	var page User
	err = res.Decode(&page)
	if err != nil {
		return errors.Wrap(err, "decode page user")
	}

	res, err = session.Get(fmt.Sprintf("/%s/conversations", config.FB.PageId), fb.Params{})
	if err != nil {
		return errors.Wrap(err, "fb.GET /pageId/conversations")
	}

	// create a paging structure.
	paging, _ := res.Paging(&session)
	noMore := false
	pageNumber := 1
	for !noMore {
		log.Infof("Process page %d", pageNumber)
		pageNumber++
		convResult := ConversationResult{}
		err = paging.Decode(&convResult)
		if err != nil {
			log.Warn("Decode conversations: ", err)
			noMore, _ = paging.Next()
			continue
		}

		var senders []User
		for _, conversation := range convResult.Data {
			convLog := log.WithField("conversationId", conversation.ID)
			userSession := fb.Session{}
			userSession.SetAccessToken(accessToken)
			res, err = userSession.Get(fmt.Sprintf("/%s", conversation.ID), fb.Params{"fields": "participants"})
			if err != nil {
				convLog.Warn("fb.GET /conversationId: ", err)
			}
			result := ParticipantsResult{}
			err = res.Decode(&result)
			if err != nil {
				convLog.Warn("decode participants: ", err)
			}
			for _, user := range result.Participants["data"] {
				if user.ID == page.ID { // Ignore page id itself
					continue
				}
				user.LastSend, err = util.ToSqlTime(conversation.UpdatedTime)
				if err != nil {
					convLog.Warn("converting time: ", err)
					user.LastSend = time.Now().Format("2006-01-02 03:04:05")
				}
				senders = append(senders, user)
			}
		}
		if len(senders) > 0 {
			err = tmpl.Execute(output, senders)
			if err != nil {
				log.Warn("render template to output: ", err)
			}
		}

		noMore, _ = paging.Next()
	}
	log.Info("DONE")
	return nil
}
