package main

import (
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/gin-gonic/gin"
	"github.com/gocolly/colly"
)

type ScrapeRequest struct {
	URL        string   `json:"url"`
	Queries    []string `json:"queries"`
	Proxies    []string `json:"proxies"`
	Cookies    []string `json:"cookies"`
	UserAgents []string `json:"userAgents"`
}

type ScrapeResult struct {
	Data []string `json:"data"`
}

func main() {
	router := gin.Default()

	// 配置静态文件服务
	router.Static("/static", "../frontend")
	router.LoadHTMLFiles("../frontend/index.html")

	router.GET("/", func(c *gin.Context) {
		c.HTML(http.StatusOK, "index.html", nil)
	})

	router.POST("/scrape", func(c *gin.Context) {
		var req ScrapeRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		collector := colly.NewCollector()

		// 设置代理
		if len(req.Proxies) > 0 {
			rand.Seed(time.Now().Unix())
			proxy := req.Proxies[rand.Intn(len(req.Proxies))]
			collector.SetProxy(proxy)
		}

		// 设置User-Agent
		if len(req.UserAgents) > 0 {
			rand.Seed(time.Now().Unix())
			userAgent := req.UserAgents[rand.Intn(len(req.UserAgents))]
			collector.UserAgent = userAgent
		}

		// 设置Cookie
		if len(req.Cookies) > 0 {
			rand.Seed(time.Now().Unix())
			cookieStr := req.Cookies[rand.Intn(len(req.Cookies))]
			cookies := strings.Split(cookieStr, ";")
			for _, cookie := range cookies {
				parts := strings.SplitN(cookie, "=", 2)
				if len(parts) == 2 {
					collector.OnRequest(func(r *colly.Request) {
						r.Headers.Set("Cookie", parts[0]+"="+parts[1])
					})
				}
			}
		}

		data := []string{}

		collector.OnHTML("body", func(e *colly.HTMLElement) {
			// 遍历请求中的每一个查询条件
			for _, query := range req.Queries {
				// 在当前HTML元素中查找匹配查询条件的所有元素
				e.DOM.Find(query).Each(func(_ int, s *goquery.Selection) {
					// 初始化一个空字符串来存储匹配元素的内容
					item := ""

					// 获取匹配元素的文本内容
					text := s.Text()
					if text != "" {
						// 如果文本内容不为空，添加到item字符串中
						item += "Text: " + text + " "
					}

					// 遍历匹配元素的所有属性
					for _, attr := range s.Nodes[0].Attr {
						// 将每个属性的名称和值添加到item字符串中
						item += strings.Title(attr.Key) + ": " + attr.Val + " "
					}

					// 特别处理链接属性，如果存在href属性，将其单独提取出来
					if href, exists := s.Attr("href"); exists {
						item += "Href: " + href + " "
					}

					// 去除item字符串首尾的空格
					item = strings.TrimSpace(item)

					// 如果item字符串不为空，将其添加到data切片中
					if item != "" {
						data = append(data, item)
					}
				})
			}
		})

		err := collector.Visit(req.URL)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, ScrapeResult{
			Data: data,
		})
	})

	router.Run(":8080")
}
