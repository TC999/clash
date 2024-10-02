package route

import (
	"context"
	"math"
	"net/http"

	"github.com/doreamon-design/clash/component/resolver"
	"github.com/go-zoox/zoox"

	"github.com/miekg/dns"
	"github.com/samber/lo"
)

func queryDNS(ctx *zoox.Context) {
	if resolver.DefaultResolver == nil {
		ctx.JSON(http.StatusInternalServerError, newError("DNS section is disabled"))
		return
	}

	name := ctx.Query().Get("name").String()
	qTypeStr, _ := lo.Coalesce(ctx.Query().Get("type").String(), "A")

	qType, exist := dns.StringToType[qTypeStr]
	if !exist {
		ctx.JSON(http.StatusBadRequest, newError("invalid query type"))
		return
	}

	c, cancel := context.WithTimeout(context.Background(), resolver.DefaultDNSTimeout)
	defer cancel()

	msg := dns.Msg{}
	msg.SetQuestion(dns.Fqdn(name), qType)
	resp, err := resolver.DefaultResolver.ExchangeContext(c, &msg)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, newError(err.Error()))
		return
	}

	responseData := zoox.H{
		"Status":   resp.Rcode,
		"Question": resp.Question,
		"TC":       resp.Truncated,
		"RD":       resp.RecursionDesired,
		"RA":       resp.RecursionAvailable,
		"AD":       resp.AuthenticatedData,
		"CD":       resp.CheckingDisabled,
	}

	rr2Json := func(rr dns.RR, _ int) zoox.H {
		header := rr.Header()
		return zoox.H{
			"name": header.Name,
			"type": header.Rrtype,
			"TTL":  header.Ttl,
			"data": lo.Substring(rr.String(), len(header.String()), math.MaxUint),
		}
	}

	if len(resp.Answer) > 0 {
		responseData["Answer"] = lo.Map(resp.Answer, rr2Json)
	}
	if len(resp.Ns) > 0 {
		responseData["Authority"] = lo.Map(resp.Ns, rr2Json)
	}
	if len(resp.Extra) > 0 {
		responseData["Additional"] = lo.Map(resp.Extra, rr2Json)
	}

	ctx.JSON(http.StatusOK, responseData)
}
