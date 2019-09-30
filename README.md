# ssmsh
ssmsh is an interactive shell for the EC2 Parameter Store. Features:
* Interact with the parameter store hierarchy using familiar commands like cd, ls, cp, mv, and rm
* Supports relative paths and shorthand (`..`) syntax
* Operate on parameters between regions
* Recursively list, copy, and remove parameters
* Get parameter history
* Create new parameters using put
* Advanced parameters (with policies)
* Supports emacs-style command shell navigation hotkeys
* Submit batch commands with the `-file` flag
* Inline commands


## Installation

1. Download [here](https://github.com/bwhaley/ssmsh/releases) or clone and build from this repo.
2. Set up [AWS credentials](http://docs.aws.amazon.com/sdk-for-go/v1/developer-guide/configuring-sdk.html#specifying-credentials).

## Configuration

You can set up a `.ssmshrc` to configure `ssmsh`. By default, `ssmsh` will load `~/.ssmshrc` if it exists. Use the `-config` argument to set a different path.

```bash
[default]
type=SecureString
overwrite=true
decrypt=true
profile=my-profile
region=us-east-1
key=3example-89a6-4880-b544-73ad3db2ff3b
```

A few notes on configuration:
* When setting the region, the `AWS_REGION` env var takes top priority, followed by the setting in `.ssmshrc`, followed by the value set in the AWS profile (if configured)
* When setting the profile, the `AWS_PROFILE` env var takes top priority, followed by the setting in `.ssmshrc`
* If you set a KMS key, it will only work in the region where that key is located. You can use the `key` command while in the shell to change the key.

## Usage
### Help
```bash
/> help

Commands:
cd           change your relative location within the parameter store
clear        clear the screen
cp           copy source to dest
decrypt      toggle parameter decryption
exit         exit the program
get          get parameters
help         display help
history      get parameter history
key          set the KMS key
ls           list parameters
mv           move parameters
policy       create named parameter policy
profile      switch to a different AWS IAM profile
put          set parameter
region       change region
rm           remove parameters
```

### List contents of a path
Note: Listing a large number of parameters may take a long time because the maximum number of results per API call is 10. Press ^C to interrupt if a listing is taking too long. Example usage:
```bash
/> ls
dev/
/> ls -r
/dev/app/url
/dev/db/password
/dev/db/username
/> ls /dev/app
url
/>
```

### Change dir and list from current working dir
```bash
/> cd /dev
/dev> ls
app/
db/
/dev>
```

### Get a parameter
```bash
/> get /dev/db/username
[{
  ARN: "arn:aws:ssm:us-east-1:012345678901:parameter/dev/db/username",
  LastModifiedDate: 2019-09-29 23:22:19 +0000 UTC,
  Name: "/dev/db/username",
  Type: "SecureString",
  Value: "foo",
  Version: 1
}]
/> cd /dev/db
/dev/db> get ../app/url
[{
  ARN: "arn:aws:ssm:us-east-1:318677964956:parameter/dev/app/url",
  LastModifiedDate: 2019-09-29 23:22:49 +0000 UTC,
  Name: "/dev/app/url",
  Type: "SecureString",
  Value: "https://www.example.com",
  Version: 1
}]
/dev/db>
```

### Toggle decryption for SecureString parameters
```bash
/> decrypt
Decrypt is false
/> decrypt true
Decrypt is true
/>
```

### Get parameter history
```bash
/> history /dev/app/url
[{
  KeyId: "alias/aws/ssm",
  Labels: [],
  LastModifiedDate: 2019-09-29 23:22:49 +0000 UTC,
  LastModifiedUser: "arn:aws:iam::318677964956:root",
  Name: "/dev/app/url",
  Policies: [],
  Tier: "Standard",
  Type: "SecureString",
  Value: "https://www.example.com",
  Version: 1
}]
```

### Copy a parameter
```bash
/> cp /dev/app/url /test/app/url
/> ls -r /dev/app /test/app
/dev/app:
/dev/app/url
/test/app:
/test/app/url
```

### Copy an entire hierarchy
```bash
/> cp -r /dev /test
/> ls -r /test
/test/app/url
/test/db/password
/test/db/username
```

### Remove parameters
```bash
/> rm /test/app/url
/> ls -r /test
/test/db/password
/test/db/username
/> rm -r /test
/> ls -r /test
/>
```

### Put new parameters
```bash
Multiline:
/> put
Input options. End with a blank line.
... name=/dev/app/domain
... value="www.example.com"
... type=String
... description="The domain of the app in dev"
...
/>
```
Single line version:

```bash
/> put name=/dev/app/domain value="www.example.com" type=String description="The domain of the app in dev"
```

### Advanced parameters with policies
Use [parameter policies](https://docs.aws.amazon.com/systems-manager/latest/userguide/parameter-store-policies.html) to do things like expire (automatically delete) parameters at a specified time:
```bash
/> policy urlExpiration Expiration(Timestamp=2013-03-31T21:00:00.000Z)
/> policy ReminderPolicy ExpirationNotification(Before=30,Unit=days) NoChangeNotification(After=7,Unit=days)
/> put name=/dev/app/url value="www.example.com" type=String policies=[urlExpiration,ReminderPolicy]
```

### Switch AWS profile
Switches to another profile as configured in `~/.aws/config`.
```bash
/> profile
default
/> profile project1
/> profile
project1
```

### Change active region
```bash
/> region eu-central-1
/> region
eu-central-1
/>
```

### Operate on other regions
A few examples of working with regions.
```bash
/> put region=eu-central-1  name=/dev/app/domain value="www.example.com" type=String description="The domain of the app in dev"
/> cp -r us-east-1:/dev us-west-2:/dev
/> ls -r us-west-2:/dev
/> region us-east-2
/> get us-west-2:/dev/db/username us-east-1:/dev/db/password
```

###  Read commands in batches
```bash
$ cat << EOF > commands.txt
put name=/dev/app/domain value="www.example.com" type=String description="The domain of the app in dev"
rm /dev/app/domain
cp -r /dev /test
EOF
$ ssmsh -file commands.txt
$ cat commands.txt | ssmsh -file -  # Read commands from STDIN
```

###  Inline commands
```
$ ssmsh put name=/dev/app/domain value="www.example.com" type=String description="The domain of the app in dev"
```

## todo (maybe)
* [ ] Flexible and improved output formats
* [ ] Release via homebrew
* [ ] Copy between accounts using profiles
* [ ] Find parameter
* [ ] Integration w/ CloudWatch Events for scheduled parameter updates
* [ ] Export/import
* [ ] Support globbing and/or regex
* [ ] In memory parameter cache
* [ ] Read parameters as local env variables


## License
MIT

## Contributing/compiling
1. Ensure you have at least go v1.12
```
$ go version
go version go1.12 linux/amd64
```
2. Ensure your `$GOPATH` exists and is in your `$PATH`
```
export GOPATH=$HOME/go
export PATH=$PATH:$GOROOT/bin:$GOPATH/bin
```
3. Run `go get github.com/bwhaley/ssmsh`
4. Run `cd $GOPATH/src/github.com/bwhaley/ssmsh && make` to build and install the binary to `$GOPATH/bin/ssmsh`


## Related tools
Tool | Description
---- | -----------
[Chamber](https://github.com/segmentio/chamber) | A tool for managing secrets
[Parameter Store Manager](https://github.com/smblee/parameter-store-manager) | A GUI for working with the Parameter Store

## Credits
Library | Use
------- | -----
[abiosoft/ishell](https://github.com/abiosoft/ishell) | The interactive shell for golang
[aws-sdk-go](https://github.com/aws/aws-sdk-go) | The AWS SDK for Go
[mattn/go-shellwords](github.com/mattn/go-shellwords) | Parsing for the shell made easy
