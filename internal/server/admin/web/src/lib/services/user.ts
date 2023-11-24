export const getLoggedInUser = async () => {
  const response = await fetch("/api/users/me");
  return await response.json();
};
