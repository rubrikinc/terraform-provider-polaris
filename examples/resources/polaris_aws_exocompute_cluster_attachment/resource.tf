resource "polaris_aws_exocompute_cluster_attachment" "attachment" {
  cluster_name  = "my-eks-cluster"
  exocompute_id = polaris_aws_exocompute.exocompute.id
}

resource "kubernetes_manifest" "exocompute" {
  manifest = yamldecode(
    polaris_aws_exocompute_cluster_attachment.attachment.k8s_manifest
  )
}
