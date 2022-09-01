package types

import "strings"

// BrowseNodeInfo represents BrowseNodeInfo object in json response
type BrowseNodeInfo struct {
	BrowseNodes      []BrowseNode     `json:"BrowseNodes,omitempty"`
	WebsiteSalesRank WebsiteSalesRank `json:"WebsiteSalesRank,omitempty"`
}

type BrowserNodeExtract struct {
	Crumb             string
	SalesRank         int
	SalesRankCategory string
	CategoryPath      string
	CategoryRank      int
}

func (b BrowseNodeInfo) ExtractInfo() *BrowserNodeExtract {
	res := &BrowserNodeExtract{}
	for idx, browserNode := range b.BrowseNodes {
		nodeInfo := browserNode.ExtractInfo()
		if idx == 0 {
			res = nodeInfo
		}
		if nodeInfo.SalesRank != 0 {
			res = nodeInfo
			break
		}
	}
	res.SalesRankCategory = b.WebsiteSalesRank.ContextFreeName
	res.SalesRank = b.WebsiteSalesRank.SalesRank
	return res
}

// BrowseNode represents BrowseNode object in json response
type BrowseNode struct {
	Ancestor        BrowseNodeAncestor `json:"Ancestor,omitempty"`
	Children        []BrowseNodeChild  `json:"Children,omitempty"`
	ContextFreeName string             `json:"ContextFreeName,omitempty"`
	DisplayName     string             `json:"DisplayName,omitempty"`
	ID              string             `json:"Id,omitempty"`
	IsRoot          bool               `json:"IsRoot,omitempty"`
	SalesRank       int                `json:"SalesRank,omitempty"`
}

// GetCrumb get crumb value in sem3 format
func (b BrowseNode) ExtractInfo() *BrowserNodeExtract {
	crumbArr := []string{}
	ctxFreeArr := []string{}
	crumbArr = append(crumbArr, b.DisplayName)
	ctxFreeArr = append(ctxFreeArr, b.ContextFreeName)
	ancestor := b.Ancestor

	res := &BrowserNodeExtract{
		CategoryRank: b.SalesRank,
		CategoryPath: b.ContextFreeName,
	}

	for ancestor.DisplayName != "" {
		if ancestor.DisplayName != "Categories" {
			crumbArr = append([]string{ancestor.DisplayName}, crumbArr...)
			ctxFreeArr = append([]string{ancestor.ContextFreeName}, ctxFreeArr...)
		}
		if ancestor.Ancestor == nil {
			break
		}
		ancestor = *ancestor.Ancestor
	}

	// check to match crumb with value at amazon.com
	if len(crumbArr) > 2 {
		if crumbArr[0] == ctxFreeArr[1] {
			crumbArr[1] = ctxFreeArr[1]
			crumbArr = crumbArr[1:]
		}
	}

	res.Crumb = strings.Join(crumbArr, "|")
	return res
}

// BrowseNodeAncestor represents BrowseNodeAncestor object in json response
type BrowseNodeAncestor struct {
	Ancestor        *BrowseNodeAncestor `json:"Ancestor,omitempty"`
	ContextFreeName string              `json:"ContextFreeName,omitempty"`
	DisplayName     string              `json:"DisplayName,omitempty"`
	ID              string              `json:"Id,omitempty"`
}

// BrowseNodeChild represents BrowseNodeChild object in json response
type BrowseNodeChild struct {
	ContextFreeName string `json:"ContextFreeName,omitempty"`
	DisplayName     string `json:"DisplayName,omitempty"`
	ID              string `json:"Id,omitempty"`
}

// WebsiteSalesRank represents WebsiteSalesRank object in json response
type WebsiteSalesRank struct {
	ContextFreeName string `json:"ContextFreeName,omitempty"`
	DisplayName     string `json:"DisplayName,omitempty"`
	ID              string `json:"Id,omitempty"`
	SalesRank       int    `json:"SalesRank,omitempty"`
}
