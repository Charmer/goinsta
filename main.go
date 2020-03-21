package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"io/ioutil"
	"math/rand"
	"net/http"
	"strconv"
	"sync"
	"time"
)

type jsonAccount struct {
	Data struct {
		User struct {
			EdgeOwnerToTimelineMedia struct {
				Count    int `json:"count"`
				PageInfo struct {
					HasNextPage bool   `json:"has_next_page"`
					EndCursor   string `json:"end_cursor"`
				} `json:"page_info"`
				Edges []struct {
					Node struct {
						ID                 string `json:"id"`
						Typename           string `json:"__typename"`
						EdgeMediaToCaption struct {
							Edges []interface{} `json:"edges"`
						} `json:"edge_media_to_caption"`
						Shortcode          string `json:"shortcode"`
						EdgeMediaToComment struct {
							Count int `json:"count"`
						} `json:"edge_media_to_comment"`
						CommentsDisabled bool `json:"comments_disabled"`
						TakenAtTimestamp int  `json:"taken_at_timestamp"`
						Dimensions       struct {
							Height int `json:"height"`
							Width  int `json:"width"`
						} `json:"dimensions"`
						DisplayURL           string `json:"display_url"`
						EdgeMediaPreviewLike struct {
							Count int `json:"count"`
						} `json:"edge_media_preview_like"`
						Owner struct {
							ID string `json:"id"`
						} `json:"owner"`
						ThumbnailSrc       string `json:"thumbnail_src"`
						ThumbnailResources []struct {
							Src          string `json:"src"`
							ConfigWidth  int    `json:"config_width"`
							ConfigHeight int    `json:"config_height"`
						} `json:"thumbnail_resources"`
						IsVideo bool `json:"is_video"`
					} `json:"node,omitempty"`
				} `json:"edges"`
			} `json:"edge_owner_to_timeline_media"`
		} `json:"user"`
	} `json:"data"`
	Status string `json:"status"`
}

type Media struct {
	ID               string
	CommentsDisabled bool
	Video            bool
	CreatedAt        string
	CommentsCount    int
	LikesCount       int
	Dimensions       struct {
		w int
		h int
	}
	URL string
}

type Profile struct {
	ID        int    `db:"id"`
	URL       string `db:"url"`
	UserID    string `db:"user_id"`
	CreatedAt string `db:"created_at"`
	UpdatedAt string `db:"updated_at"`
	Status    string `db:"Status"`
	InstID    int `db:"inst_id"`
}

func main() {
	var wg sync.WaitGroup
	db, err := sql.Open("mysql", "root:root@/instat")
	if err != nil {
		panic(err)
	}
	defer db.Close()
	rows, err := db.Query("select * from instat.profiles")
	if err != nil {
		panic(err)
	}
	defer rows.Close()
	profiles := []Profile{}

	for rows.Next() {
		p := Profile{}
		err := rows.Scan(&p.ID, &p.UserID, &p.CreatedAt, &p.UpdatedAt, &p.URL, &p.Status, &p.InstID)
		if err != nil {
			fmt.Println(err)
			continue
		}
		profiles = append(profiles, p)
	}

	for _, p := range profiles {
		wg.Add(1)
		go Parser(p.InstID)
	}
	time.Sleep(time.Duration(60) * time.Second)
}

func Parser(UserId int) {
	url := "https://www.instagram.com/graphql/query/?query_id=17888483320059182&id=" + strconv.Itoa(UserId) + "&first=50"
	EndCursor := ""
	medias := make(map[string]Media)
	for true {
		spaceClient := http.Client{
			Timeout: time.Second * 2, // Maximum of 2 secs
		}

		fmt.Println("Getting data from " + url + "&after=" + EndCursor)
		req, err := http.NewRequest(http.MethodGet, url+"&after="+EndCursor, nil)
		if err != nil {
			panic(err)
		}
		req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_13_6) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/13.0 Safari/605.1.15")
		res, getErr := spaceClient.Do(req)
		if getErr != nil {
			panic(err)
		}

		body, readErr := ioutil.ReadAll(res.Body)
		if readErr != nil {
			panic(err)
		}
		fmt.Println("Data loaded!")

		account1 := jsonAccount{}
		jsonErr := json.Unmarshal(body, &account1)
		if account1.Status == "fail" {
			fmt.Println("FAILED!")
			break
		}
		if jsonErr != nil {
			panic(err)
		}
		for k := range account1.Data.User.EdgeOwnerToTimelineMedia.Edges {
			m := Media{
				ID:               account1.Data.User.EdgeOwnerToTimelineMedia.Edges[k].Node.ID,
				CommentsDisabled: account1.Data.User.EdgeOwnerToTimelineMedia.Edges[k].Node.CommentsDisabled,
				Video:            account1.Data.User.EdgeOwnerToTimelineMedia.Edges[k].Node.IsVideo,
				CommentsCount:    account1.Data.User.EdgeOwnerToTimelineMedia.Edges[k].Node.EdgeMediaToComment.Count,
				LikesCount:       account1.Data.User.EdgeOwnerToTimelineMedia.Edges[k].Node.EdgeMediaPreviewLike.Count,
				CreatedAt:        strconv.Itoa(account1.Data.User.EdgeOwnerToTimelineMedia.Edges[k].Node.TakenAtTimestamp),
				Dimensions: struct {
					w int
					h int
				}{
					w: account1.Data.User.EdgeOwnerToTimelineMedia.Edges[k].Node.Dimensions.Width,
					h: account1.Data.User.EdgeOwnerToTimelineMedia.Edges[k].Node.Dimensions.Height,
				},
				URL: "https://www.instagram.com/p/" + account1.Data.User.EdgeOwnerToTimelineMedia.Edges[k].Node.Shortcode,
			}
			medias[m.ID] = m
			fmt.Println(len(medias))
		}
		if account1.Data.User.EdgeOwnerToTimelineMedia.PageInfo.HasNextPage {
			EndCursor = account1.Data.User.EdgeOwnerToTimelineMedia.PageInfo.EndCursor
		} else {
			break
		}
		r := rand.Intn(5)
		time.Sleep(time.Duration(r) * time.Second)
	}

	//for _, media := range medias {
	//	fmt.Println(media.URL)
	//}
	fmt.Println(len(medias))
}