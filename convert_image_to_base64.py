import base64
import json

# Convert image to Base64
with open("formula1.png", "rb") as image_file:
    base64_string = base64.b64encode(image_file.read()).decode("utf-8")

# Create a dictionary with the required JSON structure
data = {
    "file_name": "formula1.png",
    "image_base64": base64_string
}

# Save as JSON file
with open("payload.json", "w") as json_file:
    json.dump(data, json_file, indent=4)