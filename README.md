# ssmsh
ssmsh is an interactive shell for the EC2 Parameter Store. Features:
* Interact with the parameter store hierarchy using familiar commands like cd, ls, cp, mv, and rm
* Supports relative paths and shorthand (`..`) syntax
* Operate on parameters between regions
* Recursively list, copy, and remove parameters
* Get parameter history
* Create new parameters using put
* Supports emacs-style command shell navigation hotkeys
* Submit batch commands with the `-file` flag
* Inline commands


## Installation

1. Download [here](https://github.com/kountable/ssmsh/releases) or clone and build from this repo.
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
  region       change region
  rm           remove parameters
```
### List contents of a path
```bash
/>ls /House
Lannister/
Stark/
Targaryen/
```

### Change dir and list from current working dir
```bash
/>cd /House
/House>ls
Lannister/
Stark/
Targaryen/
```

### Get parameter
```bash
/>cd /House/Stark
/House/Stark>get JonSnow
[{
  Name: "/House/Stark/JonSnow",
  Type: "String",
  Value: "Bastard",
  Version: 2
}]
```

### Get encrypted parameters
```bash
/>cd /House/Stark
/House/Stark>get VerySecretInformation
[{
  Name: "/House/Stark/VerySecretInformation",
  Type: "SecureString",
  Value: "AQICAHhBW4N+....",
  Version: 1
}]
/House/Stark>decrypt
Decrypt is true
/House/Stark>get VerySecretInformation
[{
  Name: "/House/Stark/VerySecretInformation",
  Type: "SecureString",
  Value: "The three-eyed raven lives.",
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
/> cp /House/Stark/SansaStark /House/Lannister/SansaStark
```

### Copy an entire hierarchy
```bash
/> cp -R /House/Stark /House/Targaryen
```

### Remove parameters
```bash
/> rm /House/Stark/EddardStark
/> cd /House/Stark
/House/Stark> rm -r ../Lannister
```

### Put new parameters
```bash
/> put
Input options. End with a blank line.
... name=/House/Targaryen/DaenerysTargaryen
... value="Khaleesi"
... type=String
... description="Mother of Dragons"
...
/>
```
Alternatively:

```bash
/> put name=/House/Targaryen/DaenerysTargaryen value="Khaleesi" type=String description="Mother of Dragons"
```

### Change active region
```bash
/> region eu-central-1
/> region
eu-central-1
/>
```

### Operate on other regions

```bash
/> put region=eu-central-1 name=/House/Targaryen/DaenerysTargaryen value="Khaleesi" type=String description="Mother of Dragons"
/> cp -r us-west-2:/House/Stark/ eu-central-1:/House/Targaryen
/> get eu-central-1:/House/Stark/JonSnow us-west-2:/House/Stark/JonSnow
```

###  Read commands in batches
```bash
$ cat << EOF > commands.txt
put name=/House/Targaryen/DaenerysTargaryen value="Khaleesi" type=String description="Mother of Dragons"
rm /House/Stark/RobStark
cp -R /House/Baratheon /House/Lannister
EOF
$ ssmsh -file commands.txt
$ cat commands.txt | ssmsh -file -  # Read commands from STDIN
```

###  Inline commands
```
$ ssmsh put name=/House/Lannister/CerseiLannister value="Noble" description="Daughter of Tywin" type=string
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
* [ ] Read parameters as local env variables


## License
MIT

## Contributing/compiling
1. Ensure you have at least go v1.10
```
$ go version
go version go1.10 linux/amd64
```
2. Ensure your `$GOPATH` exists and is in your `$PATH`
```
export GOPATH=$HOME/go
export PATH=$PATH:$GOROOT/bin:$GOPATH/bin
```
3. Run `go get github.com/kountable/ssmsh` 
4. Run `cd $GOPATH/src/github.com/kountable/ssmsh && make` to build and install the binary to `$GOPATH/bin/ssmsh`


## Related tools
[Chamber](https://github.com/segmentio/chamber) | A tool for managing secrets

## Credits
Library | Use
------- | -----
[abiosoft/ishell](https://github.com/abiosoft/ishell) | The interactive shell for golang
[aws-sdk-go](https://github.com/aws/aws-sdk-go) | The AWS SDK for Go
[mattn/go-shellwords](github.com/mattn/go-shellwords) | Parsing for the shell made easy
