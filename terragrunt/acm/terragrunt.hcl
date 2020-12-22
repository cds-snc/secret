# DO NOT CHANGE ANYTHING BELOW HERE UNLESS YOU KNOW WHAT YOU ARE DOING

terraform {
  source = "../../aws//acm"
}

include {
  path = find_in_parent_folders()
}