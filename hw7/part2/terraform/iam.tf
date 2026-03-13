# Learner Lab doesn't allow creating IAM roles.
# Use the pre-existing LabRole for both execution and task roles.

data "aws_iam_role" "lab_role" {
  name = "LabRole"
}