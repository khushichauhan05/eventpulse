"""Generate terminal-style PNG screenshots for EventPulse documentation."""

from PIL import Image, ImageDraw, ImageFont
import os

OUTPUT_DIR = os.path.join(os.path.dirname(__file__), "screenshots")

BG = (22, 27, 34)
TITLE_BG = (33, 38, 45)
BORDER = (48, 54, 61)
WHITE = (230, 237, 243)
GREEN = (87, 210, 136)
CYAN = (121, 192, 255)
YELLOW = (224, 183, 82)
MAGENTA = (210, 153, 255)
RED = (255, 123, 114)
GRAY = (139, 148, 158)
DIM = (100, 110, 120)
PROMPT_COLOR = (87, 210, 136)
CMD_COLOR = (230, 237, 243)
URL_COLOR = (121, 192, 255)

FONT_SIZE = 14
LINE_H = 22
PAD_X = 20
PAD_Y = 16
TITLE_H = 38
DOT_R = 7
DOT_GAP = 22

try:
    font = ImageFont.truetype("C:/Windows/Fonts/consola.ttf", FONT_SIZE)
    font_bold = ImageFont.truetype("C:/Windows/Fonts/consolab.ttf", FONT_SIZE)
except Exception:
    font = ImageFont.load_default()
    font_bold = font


def make_image(lines, title, width=820):
    height = TITLE_H + PAD_Y * 2 + len(lines) * LINE_H + PAD_Y
    img = Image.new("RGB", (width, height), BG)
    d = ImageDraw.Draw(img)

    # title bar
    d.rectangle([0, 0, width, TITLE_H], fill=TITLE_BG)
    d.rectangle([0, TITLE_H, width, TITLE_H + 1], fill=BORDER)

    # traffic lights
    for i, color in enumerate([(255, 95, 86), (255, 189, 46), (39, 201, 63)]):
        cx = PAD_X + i * DOT_GAP
        cy = TITLE_H // 2
        d.ellipse([cx - DOT_R, cy - DOT_R, cx + DOT_R, cy + DOT_R], fill=color)

    # title text
    d.text((width // 2, TITLE_H // 2), title, fill=GRAY, font=font, anchor="mm")

    y = TITLE_H + PAD_Y
    for segments in lines:
        x = PAD_X
        if isinstance(segments, str):
            segments = [(segments, WHITE)]
        for text, color in segments:
            d.text((x, y), text, fill=color, font=font)
            x += d.textlength(text, font=font)
        y += LINE_H

    return img


def save(img, name):
    path = os.path.join(OUTPUT_DIR, name)
    img.save(path, "PNG")
    print(f"Saved {path}")


# ── 1. api-request.png ───────────────────────────────────────────────────────
lines_api = [
    [("$ ", PROMPT_COLOR), ("curl -s -X POST http://localhost:8080/events \\", CMD_COLOR)],
    [("  ", DIM),          ("  -H ", WHITE), ('"Content-Type: application/json" \\', CYAN)],
    [("  ", DIM),          ("  -d ", WHITE), ("'", CYAN), ('{"user_id":"user_001","event_type":"purchase","amount":75000}', YELLOW), ("'", CYAN)],
    [],
    [("  {", WHITE)],
    [('    "message": ', WHITE), ('"Event Published"', GREEN)],
    [("  }", WHITE)],
    [],
    [("$ ", PROMPT_COLOR), ("curl -s -X POST http://localhost:8080/events \\", CMD_COLOR)],
    [("  ", DIM),          ("  -H ", WHITE), ('"Content-Type: application/json" \\', CYAN)],
    [("  ", DIM),          ("  -d ", WHITE), ("'", CYAN), ('{"user_id":"user_002","event_type":"transfer","amount":150000}', YELLOW), ("'", CYAN)],
    [],
    [("  {", WHITE)],
    [('    "message": ', WHITE), ('"Event Published"', GREEN)],
    [("  }", WHITE)],
    [],
    [("$ ", PROMPT_COLOR), ("curl -s http://localhost:8080/health", CMD_COLOR)],
    [],
    [("  {", WHITE)],
    [('    "status": ', WHITE), ('"ok"', GREEN)],
    [("  }", WHITE)],
]
save(make_image(lines_api, "api-gateway  —  POST /events & GET /health"), "api-request.png")


# ── 2. analytics-service.png ─────────────────────────────────────────────────
lines_an = [
    [("$ ", PROMPT_COLOR), ("docker logs analytics-service --tail 10", CMD_COLOR)],
    [],
    [('time=2026-06-18T10:30:23Z  level=', DIM), ("INFO", GREEN), ('  msg="analytics service started"', DIM)],
    [('time=2026-06-18T10:30:23Z  level=', DIM), ("INFO", GREEN), ('  msg="health server started"  port=', DIM), ("8081", CYAN)],
    [('time=2026-06-18T10:30:40Z  level=', DIM), ("INFO", GREEN), ('  msg="processed event"  user_id=', DIM), ("901", CYAN),      ("  risk_score=", DIM), ("90", YELLOW)],
    [('time=2026-06-18T10:34:46Z  level=', DIM), ("INFO", GREEN), ('  msg="processed event"  user_id=', DIM), ("904", CYAN),      ("  risk_score=", DIM), ("90", YELLOW)],
    [('time=2026-06-18T13:42:31Z  level=', DIM), ("INFO", GREEN), ('  msg="processed event"  user_id=', DIM), ("user_001", CYAN), ("  risk_score=", DIM), ("90", YELLOW)],
    [('time=2026-06-18T13:42:31Z  level=', DIM), ("INFO", GREEN), ('  msg="processed event"  user_id=', DIM), ("user_002", CYAN), ("  risk_score=", DIM), ("90", YELLOW)],
    [('time=2026-06-18T13:42:31Z  level=', DIM), ("INFO", GREEN), ('  msg="processed event"  user_id=', DIM), ("user_003", CYAN), ("  risk_score=", DIM), ("90", YELLOW)],
    [],
    [("$ ", PROMPT_COLOR), ("curl -s http://localhost:8081/health", CMD_COLOR)],
    [],
    [('  {"status":"', WHITE), ("ok", GREEN), ('"}', WHITE)],
]
save(make_image(lines_an, "analytics-service  —  Kafka Consumer Logs", width=860), "analytics-service.png")


# ── 3. alert-service.png ─────────────────────────────────────────────────────
lines_al = [
    [("$ ", PROMPT_COLOR), ("docker logs alert-service --tail 10", CMD_COLOR)],
    [],
    [('time=2026-06-18T10:30:23Z  level=', DIM), ("INFO", GREEN), ('  msg="alert service started"', DIM)],
    [('time=2026-06-18T10:30:23Z  level=', DIM), ("INFO", GREEN), ('  msg="health server started"  port=', DIM), ("8082", CYAN)],
    [('time=2026-06-18T10:30:40Z  level=', DIM), ("INFO", GREEN), ('  msg="alert generated"  user_id=', DIM), ("901",      CYAN), ("  risk_score=", DIM), ("90", RED)],
    [('time=2026-06-18T10:34:46Z  level=', DIM), ("INFO", GREEN), ('  msg="alert generated"  user_id=', DIM), ("904",      CYAN), ("  risk_score=", DIM), ("90", RED)],
    [('time=2026-06-18T13:42:31Z  level=', DIM), ("INFO", GREEN), ('  msg="alert generated"  user_id=', DIM), ("user_001", CYAN), ("  risk_score=", DIM), ("90", RED)],
    [('time=2026-06-18T13:42:31Z  level=', DIM), ("INFO", GREEN), ('  msg="alert generated"  user_id=', DIM), ("user_002", CYAN), ("  risk_score=", DIM), ("90", RED)],
    [('time=2026-06-18T13:42:31Z  level=', DIM), ("INFO", GREEN), ('  msg="alert generated"  user_id=', DIM), ("user_003", CYAN), ("  risk_score=", DIM), ("90", RED)],
    [],
    [("$ ", PROMPT_COLOR), ("curl -s http://localhost:8082/health", CMD_COLOR)],
    [],
    [('  {"status":"', WHITE), ("ok", GREEN), ('"}', WHITE)],
]
save(make_image(lines_al, "alert-service  —  Alert Generation Logs", width=860), "alert-service.png")


# ── 4. alerts-response.png ───────────────────────────────────────────────────
lines_resp = [
    [("$ ", PROMPT_COLOR), ("curl -s http://localhost:8080/alerts | python -m json.tool", CMD_COLOR)],
    [],
    [("{", WHITE)],
    [('  "Count": ', WHITE), ("10", CYAN), (",", WHITE)],
    [('  "value": [', WHITE)],
    [("    {", WHITE)],
    [('      "id": ', WHITE),         ("10", CYAN),                   (",", WHITE)],
    [('      "user_id": ',WHITE),     ('"user_003"', YELLOW),          (",", WHITE)],
    [('      "risk_score": ', WHITE), ("90", RED),                    (",", WHITE)],
    [('      "message": ', WHITE),    ('"HIGH RISK TRANSACTION DETECTED"', RED), (",", WHITE)],
    [('      "created_at": ', WHITE), ('"2026-06-18T13:42:31.700522Z"', DIM)],
    [("    },", WHITE)],
    [("    {", WHITE)],
    [('      "id": ', WHITE),         ("9", CYAN),                    (",", WHITE)],
    [('      "user_id": ',WHITE),     ('"user_002"', YELLOW),          (",", WHITE)],
    [('      "risk_score": ', WHITE), ("90", RED),                    (",", WHITE)],
    [('      "message": ', WHITE),    ('"HIGH RISK TRANSACTION DETECTED"', RED), (",", WHITE)],
    [('      "created_at": ', WHITE), ('"2026-06-18T13:42:31.669542Z"', DIM)],
    [("    },", WHITE)],
    [("    {", WHITE)],
    [('      "id": ', WHITE),         ("8", CYAN),                    (",", WHITE)],
    [('      "user_id": ',WHITE),     ('"user_001"', YELLOW),          (",", WHITE)],
    [('      "risk_score": ', WHITE), ("90", RED),                    (",", WHITE)],
    [('      "message": ', WHITE),    ('"HIGH RISK TRANSACTION DETECTED"', RED), (",", WHITE)],
    [('      "created_at": ', WHITE), ('"2026-06-18T13:42:31.556703Z"', DIM)],
    [("    }", WHITE)],
    [("    ... 7 more alerts ...", GRAY)],
    [("  ]", WHITE)],
    [("}", WHITE)],
]
save(make_image(lines_resp, "api-gateway  —  GET /alerts  (10 alerts)", width=860), "alerts-response.png")


# ── 5. architecture.png ──────────────────────────────────────────────────────
W, H = 860, 340
img = Image.new("RGB", (W, H), BG)
d = ImageDraw.Draw(img)

d.rectangle([0, 0, W, TITLE_H], fill=TITLE_BG)
d.rectangle([0, TITLE_H, W, TITLE_H + 1], fill=BORDER)
for i, color in enumerate([(255, 95, 86), (255, 189, 46), (39, 201, 63)]):
    cx = PAD_X + i * DOT_GAP
    cy = TITLE_H // 2
    d.ellipse([cx - DOT_R, cy - DOT_R, cx + DOT_R, cy + DOT_R], fill=color)
d.text((W // 2, TITLE_H // 2), "EventPulse  —  Architecture", fill=GRAY, font=font, anchor="mm")

try:
    font_big = ImageFont.truetype("C:/Windows/Fonts/consolab.ttf", 15)
    font_sm  = ImageFont.truetype("C:/Windows/Fonts/consola.ttf",  12)
except Exception:
    font_big = font
    font_sm  = font

nodes = [
    ("Client",            100,  190, CYAN),
    ("API Gateway\n:8080", 230, 190, GREEN),
    ("Kafka\nevents.raw",  360, 145, YELLOW),
    ("Analytics\nService", 490, 145, MAGENTA),
    ("Kafka\nevents.proc", 620, 145, YELLOW),
    ("Alert\nService",     750, 145, RED),
    ("PostgreSQL",         750, 240, CYAN),
    ("Kafka\nalerts",      620, 240, YELLOW),
]

BOX_W, BOX_H = 100, 48

def box_center(x, y):
    return x, y

def draw_node(d, label, cx, cy, color):
    x0, y0 = cx - BOX_W // 2, cy - BOX_H // 2
    x1, y1 = cx + BOX_W // 2, cy + BOX_H // 2
    d.rounded_rectangle([x0, y0, x1, y1], radius=6, fill=(30, 36, 44), outline=color, width=2)
    lines = label.split("\n")
    total_h = len(lines) * 17
    start_y = cy - total_h // 2 + 2
    for ln in lines:
        d.text((cx, start_y), ln, fill=color, font=font_sm, anchor="mm")
        start_y += 17

def arrow(d, x0, y0, x1, y1, color=GRAY):
    d.line([(x0, y0), (x1, y1)], fill=color, width=2)
    # arrowhead
    import math
    angle = math.atan2(y1 - y0, x1 - x0)
    size = 8
    for da in [0.45, -0.45]:
        ax = x1 - size * math.cos(angle + da)
        ay = y1 - size * math.sin(angle + da)
        d.line([(x1, y1), (ax, ay)], fill=color, width=2)

# Draw edges
edges = [
    (100, 190, 230, 190),        # Client → API GW
    (280, 175, 360, 160),        # API GW → Kafka raw
    (410, 145, 490, 145),        # Kafka raw → Analytics
    (540, 145, 620, 145),        # Analytics → Kafka proc
    (670, 145, 750, 145),        # Kafka proc → Alert
    (750, 170, 750, 215),        # Alert → Postgres
    (700, 240, 620, 240),        # Alert → Kafka alerts
    (230, 205, 230, 240),        # API GW ↓
    (230, 240, 750, 240),        # API GW → Postgres (dashed via bottom)
]

for x0, y0, x1, y1 in edges[:7]:
    arrow(d, x0, y0, x1, y1, GRAY)

# API GW to Postgres path
d.line([(230, 205), (230, 255), (750, 255), (750, 215)], fill=DIM, width=1)

for label, cx, cy, color in nodes:
    draw_node(d, label, cx, cy, color)

# Legend
legend_y = H - 36
d.text((PAD_X, legend_y), "Client → API Gateway → Kafka → Analytics → Kafka → Alert Service → PostgreSQL", fill=GRAY, font=font_sm)

img.save(os.path.join(OUTPUT_DIR, "architecture.png"), "PNG")
print(f"Saved architecture.png")
print("All screenshots generated.")
