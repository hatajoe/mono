# slack-reactions

fetch reacted slack channel history.

```
% SLACK_API_TOKEN=XXXXX ./slack-reactions -c "alert" -r "noisy"
```

- c: channel name (without #)
- r: reaction name (without :)
- f: from date ISO8601 (optional, default: 3 days ago)
