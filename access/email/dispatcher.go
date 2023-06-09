package email

import (
    "SinguGPT/errors"
    "fmt"
    "log"
    "time"

    "github.com/google/uuid"

    "SinguGPT/access"
    "SinguGPT/access/email/imap"
    "SinguGPT/access/email/smtp"
    "SinguGPT/models"
    "SinguGPT/store"
)

func getErrorMsg(err any) (userMsg string, errMsg string) {
    switch err.(type) {
    case errors.ProgramError:
        _err := err.(errors.ProgramError)
        userMsg = "服务器异常"
        errMsg = _err.Error()
    case errors.NormalError:
        _err := err.(errors.NormalError)
        userMsg = _err.Error()
    default:
        userMsg = "服务器异常"
        errMsg = fmt.Sprintf("%v", err)
    }
    return
}

type Dispatcher struct {
    smtpClient *smtp.Client
    imapClient *imap.Client

    mails        chan *imap.Mail
    errorChannel chan error

    messageHandler access.MessageHandler
}

func (d *Dispatcher) OnMessageReceive(handler access.MessageHandler) error {
    d.messageHandler = handler
    return nil
}

func (d *Dispatcher) OnBatchUserChange(_ access.BatchUserChangeHandler) error {
    // operation not supported
    return nil
}

func (d *Dispatcher) Listen() error {
    d.smtpClient = smtp.NewSmtpClient(
        store.Config.Email.SMTP.Host,
        store.Config.Email.SMTP.Port,
        store.Config.Email.SMTP.UserName,
        store.Config.Email.SMTP.Password,
        store.Config.App.Name,
        fmt.Sprintf("[%s] 响应", store.Config.App.Name),
    )
    d.imapClient = imap.NewClient(&imap.EmailConfig{
        Host:     store.Config.Email.IMAP.Host,
        Port:     store.Config.Email.IMAP.Port,
        Username: store.Config.Email.IMAP.UserName,
        Password: store.Config.Email.IMAP.Password,
        Debug:    false,
    })
    err := d.imapClient.Listen(d.mails, d.errorChannel, 5*time.Second)
    if err != nil {
        return err
    }
    go func() {
        for {
            select {
            case err := <-d.errorChannel:
                if err != nil {
                    log.Printf("[ERROR] %v", err)
                }
            case mail := <-d.mails:
                go func() {
                    requestId := "<UNKNOWN>"
                    defer func() {
                        if err := recover(); err != nil {
                            userMsg, errMsg := getErrorMsg(err)
                            log.Printf("[ERROR]--- %s %s%s", requestId, userMsg, errMsg)
                        }
                    }()
                    email := mail.From[0][0]
                    user := store.FindUser(email)
                    if user == nil {
                        log.Printf("[WARNING] 邮箱用户 %s<%s> 不是有效用户，跳过\n", mail.From[0][1], mail.From[0][0])
                        return
                    }
                    requestId = uuid.NewString()
                    log.Printf("[INFO]>>> %s 处理用户 %s<%s> 的请求\n", requestId, user.Name, email)
                    // 解析命令
                    commands := parseCommand(mail.Subject)
                    // 创建 Contents
                    contents := make(models.Contents, len(commands)+len(mail.Contents)+len(mail.Attaches))
                    copy(contents[:len(commands)], commands)
                    // 索引
                    index := len(commands)
                    // 处理 IMAP 接收到的邮件内容
                    for _, message := range mail.Contents {
                        // 根据消息的内容类型，转换为对应的 Content 对象
                        var content models.Content
                        switch message.Type {
                        case imap.Text:
                            content = models.NewTextContent(models.TagBody, message.Text)
                        case imap.HTML:
                            content = models.NewHTMLContent(models.TagBody, message.Text)
                        }
                        contents[index] = content
                        index = index + 1
                    }
                    // 处理 IMAP 接收到的附件
                    for _, attache := range mail.Attaches {
                        // 根据附件的内容类型，转换为对应的 Content 对象
                        var content models.Content
                        switch attache.Type {
                        case imap.Text:
                            content = models.NewTextContent(models.TagBody, string(attache.Bytes))
                        case imap.HTML:
                            content = models.NewHTMLContent(models.TagBody, string(attache.Bytes))
                        case imap.Other:
                            content = models.NewByteContent(models.TagBody, attache.Bytes)
                        }
                        contents[index] = models.NewFileContent(attache.Filename, content)
                        index = index + 1
                    }
                    resp, err := d.messageHandler(user.ID, requestId, user, contents)
                    if err != nil {
                        userMsg, errMsg := getErrorMsg(err)
                        resp = models.Contents{
                            models.NewTextContent(models.TagTitle, "ERROR"),
                            models.NewTextContent(models.TagError, userMsg),
                        }
                        log.Printf("[ERROR]--- %s %s", requestId, errMsg)
                    }
                    err = d.smtpClient.Push(user, email, resp)
                    if err != nil {
                        log.Printf("[ERROR]--- %s %v", requestId, err)
                    }
                    log.Printf("[INFO]<<< %s 用户 %s<%s> 的请求处理完成\n", requestId, user.Name, email)
                }()
            }
        }
    }()
    return nil
}

func NewDispatcher() access.Dispatcher {
    return &Dispatcher{
        mails:        make(chan *imap.Mail, 20),
        errorChannel: make(chan error, 1),
    }
}
