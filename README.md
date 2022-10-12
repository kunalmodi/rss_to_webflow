# RSS To Webflow

This is a script you can throw into a cronjob to upload RSS entries into the Webflow CMS. It's much easier to use Zapier's integration, but it doesn't work when the RSS feed doesn't have URL-safe GUIDs (for instance, Substack has the post URL as the GUID).

To use this

1. Create a CMS collection in Webflow with the relevant fields. Here is an example:

<img width="773" alt="Screen Shot 2022-10-12 at 2 49 18 PM" src="https://user-images.githubusercontent.com/791388/195454592-fc8e2e4c-37cb-4a03-87e3-cad4820b5d3a.png">

2. Create a .env file that looks something like
```
RSS_URL="https://yoursubstack.substack.com/feed"
WEBFLOW_API_KEY="<API Key from Project Settings>"
WEBFLOW_COLLECTION_ID="<The CMS Collection ID You Just Created>"
```

3. Run this script periodically

```
go run main.go
```

This ensures that every GUID in the RSS feed is only uploaded once to Webflow. So it is safe to run this script as often as you want.
