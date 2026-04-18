package attackpoint

import (
	"fmt"
	"io"
	"strings"

	"golang.org/x/net/html"
)

// DiscoverForm fetches and parses the new training form to discover field names and options.
func (c *Client) DiscoverForm() (*FormSchema, error) {
	resp, err := c.Get("/newtraining.jsp")
	if err != nil {
		return nil, fmt.Errorf("fetching training form: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("training form returned status %d", resp.StatusCode)
	}

	return ParseForm(resp.Body)
}

// formInfo tracks fields discovered within a single <form> element.
type formInfo struct {
	action string
	fields map[string]FormField
}

// ParseForm parses HTML to find the training form (the one with activitytypeid)
// and extract its fields.
func ParseForm(r io.Reader) (*FormSchema, error) {
	doc, err := html.Parse(r)
	if err != nil {
		return nil, fmt.Errorf("parsing HTML: %w", err)
	}

	// Collect all forms and their fields.
	var forms []formInfo
	var currentForm *formInfo

	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "form" {
			fi := formInfo{
				action: getAttr(n, "action"),
				fields: make(map[string]FormField),
			}
			forms = append(forms, fi)
			currentForm = &forms[len(forms)-1]

			// Walk children within this form context.
			for child := n.FirstChild; child != nil; child = child.NextSibling {
				walk(child)
			}
			currentForm = nil
			return
		}

		if currentForm != nil && n.Type == html.ElementNode {
			switch n.Data {
			case "input":
				if name := getAttr(n, "name"); name != "" {
					currentForm.fields[name] = FormField{Name: name, Type: "input"}
				}
			case "textarea":
				if name := getAttr(n, "name"); name != "" {
					currentForm.fields[name] = FormField{Name: name, Type: "textarea"}
				}
			case "select":
				if name := getAttr(n, "name"); name != "" {
					options := extractOptions(n)
					currentForm.fields[name] = FormField{Name: name, Type: "select", Options: options}
				}
				return // Don't recurse into select children.
			}
		}

		for child := n.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(doc)

	// Find the form that contains "activitytypeid" — that's the training form.
	var trainingForm *formInfo
	for i := range forms {
		if _, ok := forms[i].fields["activitytypeid"]; ok {
			trainingForm = &forms[i]
			break
		}
	}

	if trainingForm == nil {
		return nil, fmt.Errorf("could not find training form (no activitytypeid field found)")
	}

	schema := &FormSchema{
		Action: trainingForm.action,
		Fields: trainingForm.fields,
	}

	// Extract activity types from the activitytypeid select.
	if field, ok := trainingForm.fields["activitytypeid"]; ok {
		schema.ActivityTypes = field.Options
	}

	return schema, nil
}

func extractOptions(selectNode *html.Node) []SelectOption {
	var options []SelectOption
	for child := selectNode.FirstChild; child != nil; child = child.NextSibling {
		if child.Type == html.ElementNode && child.Data == "option" {
			value := getAttr(child, "value")
			label := textContent(child)
			options = append(options, SelectOption{
				Value: value,
				Label: strings.TrimSpace(label),
			})
		}
	}
	return options
}

func getAttr(n *html.Node, key string) string {
	for _, attr := range n.Attr {
		if attr.Key == key {
			return attr.Val
		}
	}
	return ""
}

func textContent(n *html.Node) string {
	var sb strings.Builder
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.TextNode {
			sb.WriteString(n.Data)
		}
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(n)
	return sb.String()
}
