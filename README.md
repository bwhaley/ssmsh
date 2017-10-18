#
pssh is an interactive shell for the AWS EC2 Parameter Store.


## Installation
go get github.com/kountable/pssh

## Usage
First, set up [AWS credentials](http://docs.aws.amazon.com/sdk-for-go/v1/developer-guide/configuring-sdk.html#specifying-credentials).

```
$ pssh
Browse the EC2 Parameter Store
/> help
Commands:
  cd           change your relative location within the parameter store
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
```


## todo
* [ ] Testing
* [ ] cp
* [ ] mv
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
