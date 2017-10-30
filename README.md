#
pssh is an interactive shell for the AWS EC2 Parameter Store.


## Installation
go get github.com/kountable/pssh

## Usage
First, set up [AWS credentials](http://docs.aws.amazon.com/sdk-for-go/v1/developer-guide/configuring-sdk.html#specifying-credentials).

### Basic features
```
/> help
  cd          change your relative location within the parameter store
  clear       clear the screen
  cp           not yet implemented
  decrypt   toggle parameter decryption
  exit         exit the program
  get          get parameters
  help        display help
  history    toggle parameter history
  ls            list parameters
  mv          not yet implemented
  put          set parameter
  rm           remove parameters
/> cd /foo
/foo> ls
bar
baz
/foo> get bar
Parameter  Type               Value
/foo/bar       SecureString    -
/>decrypt
Decrypt is true
/>get bar
Parameter  Type               Value
/foo/bar      SecureString    baz
```

###  Inline commands
```
$ pssh put name=/foo/bar value=baz type=string
```

###  Read commands from files
```
$ cat << EOF > commands.txt
put name=/foo/bar value=baz type=string
put name=foo/baz value=bar type=string
$ pssh -file commands.txt
```


## todo
* [x] cp
* [ ] tests
* [ ] mv
* [ ] recursive delete
* [ ] Read commands from a file
* [ ] Flexible and improved output formats
* [ ] Release via homebrew
* [ ] Improve README


## License
MIT


## Credits
Library | Use
------- | -----
[abiosoft/ishell](https://github.com/abiosoft/ishell) | The interactive shell for golang
[aws-sdk-go](https://github.com/aws/aws-sdk-go) | The AWS SDK for Go



## Refactoring status

Add Parameter
pssh put --name /di/test/VALUE_2 --value hoho -d test description -k alias/aws/ssm  -t SecureString

Get Parameter
pssh get /di/test/VALUE_2 -d

➜  pssh git:(refactoring_experiment) ✗ ./pssh get /di/test/VALUE_2 -d
|    PARAMETER     |     TYPE     | VALUE |
|------------------|--------------|-------|
| /di/test/VALUE_2 | SecureString | hoho  |