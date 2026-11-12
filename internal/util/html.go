package util

import (
	"strings"

	"golang.org/x/net/html"
	"golang.org/x/net/html/atom"
)

// HTMLToText converts a publisher's HTML summary into plain text.
//
// It is a port of the original implementation's FormatLineBreaks, which used
// HtmlAgilityPack. The steps are:
//
//  1. Drop every carriage return and turn every newline into a space, because
//     they mean nothing in HTML but break the block detection below.
//  2. Remove comments, and the head, script and style elements.
//  3. Unwrap span and label, which carry no meaning of their own.
//  4. Put a line break before and after every p and div that holds text of its
//     own, that is, text that is not nested deeper.
//  5. Turn every br into a line break.
//  6. Return the remaining text, trimmed.
//
// The result uses "\r\n" as the line separator, matching the original.
func HTMLToText(s string) string {
	if strings.TrimSpace(s) == "" {
		return ""
	}

	// Step 1. Newlines mean nothing in HTML but confuse the block logic.
	s = strings.ReplaceAll(s, "\r", "")
	s = strings.ReplaceAll(s, "\n", " ")

	doc, err := html.Parse(strings.NewReader(s))
	if err != nil {
		// A malformed document still yields a usable tree in most cases.
		// If parsing fails outright, fall back to stripping the tags.
		return strings.TrimSpace(stripTags(s))
	}

	// Steps 2 and 3. Remove the unwanted elements and unwrap the inline ones.
	removeElements(doc, atom.Script, atom.Style, atom.Head)
	unwrapElements(doc, atom.Span, atom.Label)

	// Step 4. A block element only earns a line break when it holds text of
	// its own.
	for _, n := range collect(doc, func(n *html.Node) bool {
		return n.DataAtom == atom.P || n.DataAtom == atom.Div
	}) {
		if !hasDirectText(n) {
			continue
		}
		insertBefore(n, textNode("\r\n"))
		insertAfter(n, textNode("\r\n"))
	}

	// Step 5. A br is a line break.
	for _, n := range collect(doc, func(n *html.Node) bool {
		return n.DataAtom == atom.Br
	}) {
		replaceNode(n, textNode("\r\n"))
	}

	// Step 6. Collect the text.
	return strings.TrimSpace(innerText(doc))
}

// removeElements deletes every node matching one of the given atoms, together
// with its subtree.
func removeElements(doc *html.Node, atoms ...atom.Atom) {
	for _, n := range collect(doc, func(n *html.Node) bool {
		for _, a := range atoms {
			if n.DataAtom == a {
				return true
			}
		}
		return false
	}) {
		if n.Parent != nil {
			n.Parent.RemoveChild(n)
		}
	}
}

// unwrapElements replaces every node matching one of the given atoms with
// its own children, so the element disappears but its content stays.
func unwrapElements(doc *html.Node, atoms ...atom.Atom) {
	for _, n := range collect(doc, func(n *html.Node) bool {
		for _, a := range atoms {
			if n.DataAtom == a {
				return true
			}
		}
		return false
	}) {
		if n.Parent == nil {
			continue
		}
		parent := n.Parent
		for c := n.FirstChild; c != nil; {
			next := c.NextSibling
			// A node must be detached before it can be moved, otherwise
			// InsertBefore rejects it.
			n.RemoveChild(c)
			parent.InsertBefore(c, n)
			c = next
		}
		parent.RemoveChild(n)
	}
}

// collect returns every node in the tree, in document order, that matches the
// predicate.
func collect(doc *html.Node, match func(*html.Node) bool) []*html.Node {
	var out []*html.Node
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if match(n) {
			out = append(out, n)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	return out
}

// hasDirectText reports whether n holds a text child of its own, as opposed
// to holding only nested elements.
func hasDirectText(n *html.Node) bool {
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		if c.Type == html.TextNode && strings.TrimSpace(c.Data) != "" {
			return true
		}
	}
	return false
}

// textNode builds a text node.
func textNode(s string) *html.Node {
	return &html.Node{
		Type: html.TextNode,
		Data: s,
	}
}

// insertBefore puts newNode directly before oldNode.
func insertBefore(oldNode, newNode *html.Node) {
	if oldNode.Parent != nil {
		oldNode.Parent.InsertBefore(newNode, oldNode)
	}
}

// insertAfter puts newNode directly after oldNode.
func insertAfter(oldNode, newNode *html.Node) {
	if oldNode.Parent != nil {
		oldNode.Parent.InsertBefore(newNode, oldNode.NextSibling)
	}
}

// replaceNode swaps oldNode for newNode.
//
// x/net/html has no ReplaceChild, so the new node is inserted at the old
// node's position and the old node is then removed.
func replaceNode(oldNode, newNode *html.Node) {
	if oldNode.Parent == nil {
		return
	}
	oldNode.Parent.InsertBefore(newNode, oldNode)
	oldNode.Parent.RemoveChild(oldNode)
}

// innerText concatenates the text of every descendant node.
func innerText(n *html.Node) string {
	var b strings.Builder
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.TextNode {
			b.WriteString(n.Data)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(n)
	return b.String()
}

// stripTags removes every tag from s without interpreting the markup. It is
// the fallback for a document that cannot be parsed at all.
func stripTags(s string) string {
	var b strings.Builder
	depth := 0
	for _, r := range s {
		switch r {
		case '<':
			depth++
		case '>':
			if depth > 0 {
				depth--
			}
		default:
			if depth == 0 {
				b.WriteRune(r)
			}
		}
	}
	return b.String()
}
