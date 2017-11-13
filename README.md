# pssh
pssh is an interactive shell for the EC2 Parameter Store. Features:
* Interact with the parameter store hierarchy using familiar commands like cd, ls, cp, mv, and rm
* Recursively list, copy, and remove parameters
* Get parameter history
* Create new parameters using put
* Supports emacs-style command shell navigation hotkeys
* Submit batch commands with the `-file` flag
* Inline commands


## Installation
1. `go get github.com/kountable/pssh`
2. Set up [AWS credentials](http://docs.aws.amazon.com/sdk-for-go/v1/developer-guide/configuring-sdk.html#specifying-credentials).

## Usage
### Help
```bash
/>help

Commands:
  cd           change your relative location within the parameter store
  clear        clear the screen
  cp           copy source to dest
  decrypt      toggle parameter decryption
  exit         exit the program
  get          get parameters
  help         display help
  history      get parameter history
  ls           list parameters
  mv           move parameters
  put          set parameter
  rm           remove parameters
```
### List contents of a path
```bash
/>ls /House
Lannister/
Stark/
Targaryen/
```

### Change dir
```bash
/>cd /House
/House>
```

### Get parameter
```bash
/>get /House/Stark/JonSnow
[{
  Name: "/House/Stark/JonSnow",
  Type: "String",
  Value: "Bastard",
  Version: 2
}]
```

### Get encrypted parameters
```bash
/>get /House/Stark/VerySecretInformation
[{
  Name: "/House/Stark/VerySecretInformation",
  Type: "SecureString",
  Value: "AQICAHhBW4N+....",
  Version: 1
}]
/>decrypt
Decrypt is true
/>get /House/Stark/VerySecretInformation
[{
  Name: "/House/Stark/VerySecretInformation",
  Type: "SecureString",
  Value: "Jon is not a bastard!",
  Version: 1
}]
```

### Get parameter history
```bash
/>history /House/Stark/JonSnow
[{
  Description: "Bastard son of Eddard",
  LastModifiedDate: 2017-11-06 23:59:02 +0000 UTC,
  LastModifiedUser: "bwhaley",
  Name: "/House/Stark/JonSnow",
  Type: "String",
  Value: "Bastard",
  Version: 1
} {
  Description: "Bastard son of Eddard Stark, man of the Night's Watch",
  LastModifiedDate: 2017-11-06 23:59:05 +0000 UTC,
  LastModifiedUser: "bwhaley",
  Name: "/House/Stark/JonSnow",
  Type: "String",
  Value: "Bastard",
  Version: 2
}]
```

### Copy a parameter
```bash
/>cp /House/Stark/JonSnow /House/Targaryen/JonSnow
```

### Copy an entire hierarchy
```bash
/> cp -R /House/Stark /House/Targaryen
```

### Remove parameters
```bash
/> rm /House/Stark/EddardStark
```

### Put new parameters
```bash
/> put
Input options. End with a blank line.
... name=/House/Targaryen/DaenerysTargaryen
... value="Queen"
... type=String
... description="Mother of Dragons and Queen of Westeros"
...
/>
```
Alternatively:

```bash
/> put name=/House/Targaryen/DaenerysTargaryen value="Queen" type=String description="Mother of Dragons and Queen of Westeros"
```

###  Read commands in batches
```bash
$ cat << EOF > commands.txt
put name=/House/Targaryen/DaenerysTargaryen value="Queen" type=String description="Mother of Dragons and Queen of Westeros"
cp /House/Stark/JonSnow /House/Targaryen/JonSnow
EOF
$ pssh -file commands.txt
```

###  Inline commands
```
$ pssh put name=/House/Lannister/CerseiLannister value="Noble" description="Daughter of Tywin" type=string
```

## todo
* [ ] Flexible and improved output formats
* [ ] Release via homebrew
* [ ] Copy between accounts using profiles
* [ ] Find parameter
* [ ] update parameter (put with fewer required fields)
* [ ] Integration w/ CloudWatch Events for scheduled parameter updates
* [ ] Export/import
* [ ] Support globbing and/or regex
* [ ] In memory parameter cache


## License
MIT

## Related tools
[Chamber](https://github.com/segmentio/chamber) | A tool for managing secrets

## Credits
Library | Use
------- | -----
[abiosoft/ishell](https://github.com/abiosoft/ishell) | The interactive shell for golang
[aws-sdk-go](https://github.com/aws/aws-sdk-go) | The AWS SDK for Go
[mattn/go-shellwords](github.com/mattn/go-shellwords) | Parsing for the shell made easy
