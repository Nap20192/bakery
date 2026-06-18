package bot

import "bakery/internal/inbound/bot/response"

// responses renders all bot messages. Rendering lives in the response
// subpackage (pure, no bot state); handlers call responses.X as before.
var responses = response.Builder{}
