package commands

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/abiosoft/ishell"
)

const policyUsage string = `
policy usage: policy <name> [policies...]
Creates a named policy object to be used when creating or update parameters.
Separate multiple policies with spaces. Prints the named policy when no policy
objects are specified. Examples:

/> policy mypolicy Expiration(Timestamp=2018-12-02T21:34:33.000Z) ExpirationNotification(Before=14,Unit=days) NoChangeNotification(After=90,Unit=days)
/> policy mypolicy
Expiration(Timestamp=2018-12-02T21:34:33.000Z)
ExpirationNotification(Before=14,Unit=days)
NoChangeNotification(After=90,Unit=days)
/> policy policy1 Expiration(Timestamp=2018-12-02T21:34:33.000Z)
/> policy policy2 ExpirationNotification(Before=14,Unit=days) NoChangeNotification(After=90,Unit=days)
/> put name=/SomeParameter value=SomeValue type=string tier=advanced policies=[policy1,policy2]

See https://docs.aws.amazon.com/systems-manager/latest/userguide/parameter-store-policies.html
`

const (
	ExpirationPolicy             = "Expiration"
	ExpirationNotificationPolicy = "ExpirationNotification"
	NoChangeNotificationPolicy   = "NoChangeNotification"
	layout                       = "2006-01-02T15:04:05Z"
)

var policies = map[string]parameterPolicies{}

type Policies interface {
	Print() string
}

type parameterPolicies struct {
	expiration             Expiration
	expirationNotification []ExpirationNotification
	noChangeNotification   []NoChangeNotification
}

type Expiration struct {
	Type       string
	Version    string
	Attributes ExpirationAttributes
}

type ExpirationAttributes struct {
	Timestamp time.Time
}

type ExpirationNotification struct {
	Type       string
	Version    string
	Attributes ExpirationNotificationAttributes
}

type ExpirationNotificationAttributes struct {
	Before int
	Unit   string
}

type NoChangeNotification struct {
	Type       string
	Version    string
	Attributes NoChangeNotificationAttributes
}

type NoChangeNotificationAttributes struct {
	After int
	Unit  string
}

func policy(c *ishell.Context) {
	if len(c.Args) == 1 {
		printPolicy(c.Args[0])
	} else if len(c.Args) > 1 {
		err := createPolicy(c.Args[0], c.Args[1:])
		if err != nil {
			shell.Printf("Error: %s\n", err)
		}
	} else {
		shell.Println(policyUsage)
	}
}

func printPolicy(policyName string) (err error) {
	var policyPrinter strings.Builder
	for pName, policy := range policies {
		if pName == policyName {
			fmt.Fprintf(&policyPrinter, "%s", policy.expiration.Print())
			for _, e := range policy.expirationNotification {
				fmt.Fprintf(&policyPrinter, "%s", e.Print())
			}
			for _, e := range policy.noChangeNotification {
				fmt.Fprintf(&policyPrinter, "%s", e.Print())
			}
		}
	}
	shell.Print(policyPrinter.String())
	return nil
}

func (exp Expiration) Print() string {
	ts := exp.Attributes.Timestamp.Format(time.RFC3339)
	var attrs []string
	attrs = append(attrs, fmt.Sprintf("Timestamp=%s", ts))
	return fmt.Sprintf("%s(%s)\n", ExpirationPolicy, strings.Join(attrs, ","))
}

func (exp ExpirationNotification) Print() string {
	before := exp.Attributes.Before
	unit := exp.Attributes.Unit
	var attrs []string
	attrs = append(attrs, fmt.Sprintf("Before=%d", before))
	attrs = append(attrs, fmt.Sprintf("Unit=%s", unit))
	return fmt.Sprintf("%s(%s)\n", ExpirationNotificationPolicy, strings.Join(attrs, ","))
}

func (nochange NoChangeNotification) Print() string {
	after := nochange.Attributes.After
	unit := nochange.Attributes.Unit
	var attrs []string
	attrs = append(attrs, fmt.Sprintf("After=%d", after))
	attrs = append(attrs, fmt.Sprintf("Unit=%s", unit))
	return fmt.Sprintf("%s(%s)\n", NoChangeNotificationPolicy, strings.Join(attrs, ","))
}

func createPolicy(policyName string, policyArgs []string) (err error) {
	var policy parameterPolicies
	for _, arg := range policyArgs {
		re := regexp.MustCompile(`^([A-Za-z]+)\(([A-z0-9-:\.,=]+)\)`)
		p := re.FindStringSubmatch(arg)
		if len(p) != 3 {
			return fmt.Errorf("unable to validate policy %s", arg)
		}
		policyType := p[1]
		policyAttributes := p[2]
		switch policyType {
		case ExpirationPolicy:
			p, err := parseExpiration(policyAttributes)
			if err != nil {
				return err
			}
			policy.expiration = *p
		case ExpirationNotificationPolicy:
			p, err := parseExpirationNotification(policyAttributes)
			if err != nil {
				return err
			}
			policy.expirationNotification = append(policy.expirationNotification, *p)
		case NoChangeNotificationPolicy:
			p, err := parseNoChangeNotification(policyAttributes)
			if err != nil {
				return err
			}
			policy.noChangeNotification = append(policy.noChangeNotification, *p)
		default:
			return fmt.Errorf("Unable to parse policy type %s with attributes %s", policyType, policyAttributes)
		}
	}
	policies[policyName] = policy
	return nil
}

// Expiration(Timestamp=2018-12-02T21:34:33.000Z)
func parseExpiration(attrArgs string) (expiration *Expiration, err error) {
	var attributes ExpirationAttributes
	parts := trim(strings.Split(attrArgs, ","))
	for _, p := range parts {
		attrArg := trim(strings.Split(p, "="))
		switch strings.ToLower(attrArg[0]) {
		case "timestamp":
			str := attrArg[1]
			attributes.Timestamp, err = time.Parse(layout, str)
			if err != nil {
				return nil, err
			}
		default:
			return nil, fmt.Errorf("invalid %s attribute %s", ExpirationPolicy, p)
		}
	}
	expiration = &Expiration{ExpirationPolicy, "1.0", attributes}
	return expiration, nil
}

// ExpirationNotification(Before=14,Unit=days)
func parseExpirationNotification(attrArgs string) (expNotification *ExpirationNotification, err error) {
	var attributes ExpirationNotificationAttributes
	parts := trim(strings.Split(attrArgs, ","))
	for _, p := range parts {
		attrArg := trim(strings.Split(p, "="))
		switch strings.ToLower(attrArg[0]) {
		case "before":
			attributes.Before, err = strconv.Atoi(attrArg[1])
			if err != nil {
				return nil, err
			}
		case "unit":
			attributes.Unit = attrArg[1]
		default:
			return nil, fmt.Errorf("invalid %s attribute %s", ExpirationNotificationPolicy, p)
		}
	}
	expNotification = &ExpirationNotification{ExpirationNotificationPolicy, "1.0", attributes}
	return expNotification, nil
}

// NoChangeNotification(After=90,Unit=days)
func parseNoChangeNotification(attrArgs string) (noChange *NoChangeNotification, err error) {
	var attributes NoChangeNotificationAttributes
	parts := trim(strings.Split(attrArgs, ","))
	for _, p := range parts {
		attrArg := trim(strings.Split(p, "="))
		switch strings.ToLower(attrArg[0]) {
		case "after":
			attributes.After, err = strconv.Atoi(attrArg[1])
			if err != nil {
				return nil, err
			}
		case "unit":
			attributes.Unit = attrArg[1]
		default:
			return nil, fmt.Errorf("invalid %s attribute %s", NoChangeNotificationPolicy, p)
		}
	}
	noChange = &NoChangeNotification{NoChangeNotificationPolicy, "1.0", attributes}
	return noChange, nil

}
