package fixrules

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/yareeh/bibdb/internal"
)

// trackingParams are query keys stripped from URL fields. Mirrors the set
// skyebot uses (utm_*, fbclid, gclid).
var trackingParams = map[string]bool{
	"utm_source":   true,
	"utm_medium":   true,
	"utm_campaign": true,
	"utm_term":     true,
	"utm_content":  true,
	"fbclid":       true,
	"gclid":        true,
}

func init() {
	Register(Rule{
		ID:          "tracking-params",
		Since:       "1.4.0",
		Severity:    AutoFix,
		Description: "Strip utm_*, fbclid, gclid tracking parameters from the url field.",
		Apply: func(e *internal.Entry) Result {
			raw := e.Get("url")
			if raw == "" {
				return Result{}
			}
			u, err := url.Parse(raw)
			if err != nil {
				return Result{}
			}
			q := u.Query()
			var stripped []string
			for k := range q {
				if trackingParams[strings.ToLower(k)] {
					stripped = append(stripped, k)
					q.Del(k)
				}
			}
			if len(stripped) == 0 {
				return Result{}
			}
			u.RawQuery = q.Encode()
			cleaned := u.String()
			e.Set("url", cleaned)
			return Result{
				Changed:  true,
				Messages: []string{fmt.Sprintf("stripped %s from url", strings.Join(stripped, ", "))},
			}
		},
	})
}
