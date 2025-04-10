# Test folder one.
resource "grafana_folder" "test_folder_one" {
  uid   = "test_folder_one"
  title = "Test Folder One"
}

# Test folder two.
resource "grafana_folder" "test_folder_two" {
  uid   = "test_folder_two"
  title = "Test Folder Two"
}
