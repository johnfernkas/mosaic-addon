"""
Applet: Clock
Summary: Simple digital clock
Description: Displays the current time
Author: Mosaic
"""

load("render.star", "render")
load("time.star", "time")

def main(config):
    timezone = config.get("timezone", "America/Indiana/Indianapolis")
    
    now = time.now().in_location(timezone)
    
    return render.Root(
        child = render.Box(
            width = 64,
            height = 32,
            color = "#000",
            child = render.Column(
                expanded = True,
                main_align = "center",
                cross_align = "center",
                children = [
                    render.Text(
                        content = now.format("3:04"),
                        font = "6x13",
                        color = "#fff",
                    ),
                    render.Text(
                        content = now.format("PM"),
                        font = "tom-thumb",
                        color = "#888",
                    ),
                ],
            ),
        ),
    )
