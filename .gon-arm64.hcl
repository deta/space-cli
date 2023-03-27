source = ["dist/space-macos-arm_darwin_arm64/space"]
bundle_id = "sh.deta.cli"

apple_id {
  username = "@env:APPLE_APP_SIGN_USERNAME"
  password = "@env:APPLE_APP_SIGN_PASSWORD"
}

sign {
  application_identity = "7033D02EC11F23C6C666B6D26DAC7CA9D439FF7F"
}