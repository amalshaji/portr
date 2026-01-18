export const getLoggedInUser = async () => {
  const response = await fetch("/api/user/me");
  return await response.json();
};
