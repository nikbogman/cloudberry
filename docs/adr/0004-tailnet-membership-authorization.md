# Tailnet membership is the entire authorization boundary

Both Control APIs treat a valid `Tailscale-User-Login` identity header as sufficient to perform any action — there is no allow-list checking *which* tailnet identities may wake or suspend the server. We chose this over adding an allow-list because this is a single-user tailnet; an allow-list would be defense-in-depth with no current threat it defends against. This is a boundary worth reopening if the tailnet is ever shared with other people or devices whose access shouldn't extend to controlling the server.
