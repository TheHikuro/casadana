import type { ReviewStatus } from "@casa-dana/api"

// The design calls these Published / Pending / Hidden; the API enum is
// approved / pending / rejected. The enum stays on the wire — this map is
// display-only.
export const REVIEW_STATUS_LABELS: Record<ReviewStatus, string> = {
  approved: "Published",
  pending: "Pending",
  rejected: "Hidden",
}

export const REVIEW_STATUSES: Array<ReviewStatus> = ["approved", "pending", "rejected"]
