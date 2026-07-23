// mediaHttpSrc — build the URL served by appshell.ServeHTTP.
// userId is required in the URL for cache correctness: media_id is only
// unique per (platform, user_id), so omitting it would let the browser
// serve one user's cached photo at another user's same-numbered id.
export function mediaHttpSrc(platform: string, profileId: number, id: number): string {
  return `/media/${platform}/${profileId}/${id}`;
}
