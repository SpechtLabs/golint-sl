import { defineNavbarConfig } from "vuepress-theme-plume";

export const navbar = defineNavbarConfig([
	{ text: "Home", link: "/", icon: "mdi:home" },

	{
		text: "Getting Started",
		icon: "mdi:rocket-launch",
		link: "/getting-started/quick",
	},

	{
		text: "Guides",
		icon: "mdi:compass",
		items: [
			{
				text: "GitHub Actions",
				link: "/guides/github-actions",
				icon: "mdi:github",
			},
			{
				text: "Pre-commit Hooks",
				link: "/guides/pre-commit",
				icon: "mdi:hook",
			},
			{
				text: "golangci-lint",
				link: "/guides/golangci-lint",
				icon: "mdi:tools",
			},
			{
				text: "Configure Analyzers",
				link: "/guides/configure-analyzers",
				icon: "mdi:tune",
			},
			{
				text: "Disable Analyzers",
				link: "/guides/disable-analyzers",
				icon: "mdi:toggle-switch-off",
			},
		],
	},

	{
		text: "Understanding",
		icon: "mdi:lightbulb",
		link: "/understanding/philosophy",
	},

	{
		text: "Reference",
		icon: "mdi:book",
		items: [
			{ text: "CLI Reference", link: "/reference/cli", icon: "mdi:terminal" },
			{
				text: "Configuration",
				link: "/reference/configuration",
				icon: "mdi:file-cog",
			},
			{
				text: "All Analyzers",
				link: "/reference/analyzers/humaneerror",
				icon: "mdi:magnify-scan",
			},
		],
	},

	{
		text: "More",
		icon: "mdi:dots-horizontal",
		items: [
			{
				text: "Download",
				link: "https://github.com/SpechtLabs/golint-sl/releases",
				target: "_blank",
				rel: "noopener noreferrer",
				icon: "mdi:download",
			},
			{
				text: "Report an Issue",
				link: "https://github.com/SpechtLabs/golint-sl/issues/new/choose",
				target: "_blank",
				rel: "noopener noreferrer",
				icon: "mdi:bug-outline",
			},
		],
	},
]);
