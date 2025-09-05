import { useState, useEffect } from "react";
import { useParams } from "react-router-dom";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
} from "@/components/ui/select";
import type { Team } from "@/types";

export default function TeamSelector() {
  const { team: currentTeamSlug } = useParams<{ team: string }>();
  const [teams, setTeams] = useState<Team[]>([]);
  const [currentTeam, setCurrentTeam] = useState<Team | null>(null);

  useEffect(() => {
    const getMyTeams = async () => {
      try {
        const response = await fetch("/api/v1/user/me/teams", {
          headers: {
            "Content-Type": "application/json",
          },
        });
        if (response.ok) {
          const teamsData = await response.json();
          setTeams(teamsData);

          // Find current team
          if (currentTeamSlug) {
            const team = teamsData.find(
              (t: Team) => t.slug === currentTeamSlug
            );
            if (team) {
              setCurrentTeam(team);
            }
          }
        }
      } catch (error) {
        console.error("Failed to fetch teams:", error);
      }
    };

    getMyTeams();
  }, [currentTeamSlug]);

  const switchTeams = (value: string) => {
    window.location.href = `/${value}/overview`;
  };

  return (
    <Select value={currentTeamSlug} onValueChange={switchTeams}>
      <SelectTrigger className="text-sm focus:ring-0 w-full">
        <div className="flex items-center space-x-2">
          <div className="h-8 w-8 rounded-full bg-muted flex items-center justify-center mr-2">
            <span className="text-sm font-medium">
              {currentTeam?.name?.slice(0, 2).toUpperCase() || "TE"}
            </span>
          </div>
          <span>{currentTeam?.name || "Select Team"}</span>
        </div>
      </SelectTrigger>
      <SelectContent>
        <div className="px-2 py-1.5 text-xs font-semibold text-muted-foreground">
          Your teams
        </div>
        {teams.map((team) => (
          <SelectItem key={team.id} value={team.slug}>
            <div className="flex items-center space-x-2">
              <div className="h-8 w-8 rounded-full bg-muted flex items-center justify-center">
                <span className="text-sm font-medium">
                  {team.name.slice(0, 2).toUpperCase()}
                </span>
              </div>
              <span>{team.name}</span>
            </div>
          </SelectItem>
        ))}
      </SelectContent>
    </Select>
  );
}
