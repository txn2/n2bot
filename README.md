# n2bot

**n2bot** receives POSTed JSON data from third party applications using webhooks (also called a web callback or HTTP push API) in order to communicate stataus to and IRC channel.

**n2bot** is configured through a set of rules associated the type of posted JSON to a corresponding template used to form the channel message.


Example Configuration:
```yaml
replacements:
  - pattern: user1
    replacement: cjimti
rules:
  - name: Gitlab Merge Request
    producer: Gitlab
    contentRule:
      key: object_kind
      equals: merge_request
    description: Gitlab merge request.
    template: "{{ .object_attributes.assignee.username }}, \x0304MERGE REQUEST\x03 #{{ .object_attributes.id }} is \x0313{{ .object_attributes.state }}\x03 for \x0307{{ .project.name }}\x03 {{ .project.web_url }} cc {{ .user.username }}"

```

### Development

Run from source:
```bash
DEBUG=true CONFIG=example.yml go run ./n2bot.go
```

- [go-ircevent] is used for IRC event handling

[go-ircevent]:https://github.com/thoj/go-ircevent
[Example Gitlab merge request]:https://docs.gitlab.com/ce/user/project/integrations/webhooks.html#merge-request-events