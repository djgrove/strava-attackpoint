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

// ParseForm parses an HTML form to extract field names, types, and select options.
func ParseForm(r io.Reader) (*FormSchema, error) {
	doc, err := html.Parse(r)
	if err != nil {
		return nil, fmt.Errorf("parsing HTML: %w", err)
	}

	schema := &FormSchema{
		Fields: make(map[string]FormField),
	}

	// Find the form and extract fields.
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		switch n.Type {
		case html.ElementNode:
			switch n.Data {
			case "form":
				if action := getAttr(n, "action"); action != "" && schema.Action == "" {
					schema.Action = action
				}
			case "input":
				name := getAttr(n, "name")
				if name != "" {
					schema.Fields[name] = FormField{
						Name: name,
						Type: "input",
					}
				}
			case "textarea":
				name := getAttr(n, "name")
				if name != "" {
					schema.Fields[name] = FormField{
						Name: name,
						Type: "textarea",
					}
				}
			case "select":
				name := getAttr(n, "name")
				if name != "" {
					options := extractOptions(n)
					field := FormField{
						Name:    name,
						Type:    "select",
						Options: options,
					}
					schema.Fields[name] = field

					// Detect the activity type select.
					nameLower := strings.ToLower(name)
					if strings.Contains(nameLower, "activitytype") || strings.Contains(nameLower, "activity_type") {
						schema.ActivityTypes = options
					}
				}
				return // Don't recurse into select children — we already extracted options.
			}
		}

		for child := n.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(doc)

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
