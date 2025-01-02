package model

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type UserInfo struct {
	Data struct {
		User struct {
			AIAgentType *string `json:"ai_agent_type"`
			Biography   string  `json:"biography"`
			BioLinks    []struct {
				Title    string `json:"title"`
				LynxURL  string `json:"lynx_url"`
				URL      string `json:"url"`
				LinkType string `json:"link_type"`
			} `json:"bio_links"`
			BiographyWithEntities struct {
				RawText  string `json:"raw_text"`
				Entities []struct {
					Hashtag *struct {
						Name string `json:"name"`
					} `json:"hashtag"`
				} `json:"entities"`
			} `json:"biography_with_entities"`
			BlockedByViewer bool   `json:"blocked_by_viewer"`
			ExternalURL     string `json:"external_url"`
			EdgeFollowedBy  struct {
				Count int `json:"count"`
			} `json:"edge_followed_by"`
			FullName                 string `json:"full_name"`
			IsBusinessAccount        bool   `json:"is_business_account"`
			ProfilePicURL            string `json:"profile_pic_url"`
			ProfilePicURLHD          string `json:"profile_pic_url_hd"`
			Username                 string `json:"username"`
			EdgeOwnerToTimelineMedia struct {
				Count    int `json:"count"`
				PageInfo struct {
					HasNextPage bool   `json:"has_next_page"`
					EndCursor   string `json:"end_cursor"`
				} `json:"page_info"`
				Edges []struct {
					Node struct {
						Typename   string `json:"__typename"`
						ID         string `json:"id"`
						Shortcode  string `json:"shortcode"`
						Dimensions struct {
							Height int `json:"height"`
							Width  int `json:"width"`
						} `json:"dimensions"`
						DisplayURL string `json:"display_url"`
						VideoURL   string `json:"video_url"`
						IsVideo    bool   `json:"is_video"`
					} `json:"node"`
				} `json:"edges"`
			} `json:"edge_owner_to_timeline_media"`
		} `json:"user"`
	} `json:"data"`
}

func GetInstagramUserInfo(username string) (*UserInfo, error) {
	url := fmt.Sprintf("https://www.instagram.com/api/v1/users/web_profile_info/?username=%s", username)

	// Create a new request using http
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		fmt.Println("Error creating request:", err)
		return nil, err
	}

	// Set headers
	req.Header.Set("User-Agent", "Mozilla/5.0 (iPhone; CPU iPhone OS 14_1 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Mobile/15E148 Instagram 142.0.0.22.109 (iPhone12,5; iOS 14_1; en_US; en-US; scale=3.00; 1242x2688; 214888322) NW/1")
	req.Header.Set("Cookie", "mid=Z3KorAALAAGScutvTJvVp7LHO1HY; ig_did=29FFFEE9-F4B8-4509-8276-E561028AE30D; ig_nrcb=1; datr=VCV1Zy3p0PsS3KHalWUzMXKM; dpr=1.5; ds_user_id=71621276538; csrftoken=MRkOvxqZctmiatHxYsgAiboAqfEaX8P0; ps_l=1; ps_n=1; wd=480x559; sessionid=71621276538%3AXbFrxPjYROHG7w%3A27%3AAYdbcJEzMyCRf1I6VvvS_-d2MVtd2ZUo0keeHsMxbw; rur=CLN\05471621276538\0541767273032:01f7c46097559c8f8aed4342d4b113fc2925364ef84c922cc52610ebef9751445bb4e3e3")
	req.Header.Set("Referer", "https://www.instagram.com/")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")

	// Send the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error making request:", err)
		return nil, err
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading response body:", err)
		return nil, err
	}

	// Unmarshal JSON into struct
	var userInfo UserInfo
	err = json.Unmarshal(body, &userInfo)
	if err != nil {
		fmt.Println("Error unmarshalling JSON:", err)
		return nil, err
	}

	// Print video URLs if available
	for _, edge := range userInfo.Data.User.EdgeOwnerToTimelineMedia.Edges {
		if edge.Node.IsVideo {
			fmt.Println("Video URL:", edge.Node.VideoURL)
		}
	}

	return &userInfo, nil
}
