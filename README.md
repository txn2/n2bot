# n2bot

**n2bot** receives POSTed JSON messages, transforms them through templates, and relays them to an IRC channel based on configuration rules.

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
```shell script
DEBUG=true CONFIG=example.yml go run ./n2bot.go
```

Build container:
```shell script
docker build --build-arg version=1.1.0 -t txn2/n2bot:1.1.0 .
docker push txn2/n2bot:1.1.0
```

Run container:
```shell script
docker run -p 8080:8080 -e IP=0.0.0.0 -v $(pwd)/example.yml:/example.yml txn2/n2bot:1.1.0
```

- **n2bot** uses [go-ircevent] for IRC event handling

[go-ircevent]:https://github.com/thoj/go-ircevent
[Example Gitlab merge request]:https://docs.gitlab.com/ce/user/project/integrations/webhooks.html#merge-request-events