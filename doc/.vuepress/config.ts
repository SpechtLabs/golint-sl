import { viteBundler } from "@vuepress/bundler-vite";
import { registerComponentsPlugin } from "@vuepress/plugin-register-components";
import { path } from "@vuepress/utils";
import { defineUserConfig } from "vuepress";
import { plumeTheme } from "vuepress-theme-plume";

export default defineUserConfig({
	base: "/",
	lang: "en-US",
	title: "golint-sl",
	description:
		"SpechtLabs best practices for writing production-ready Go code - 32 analyzers for code quality, safety, and observability",

	head: [
		[
			"meta",
			{
				name: "description",
				content:
					"A comprehensive Go linter with 32 analyzers enforcing code quality, safety, architecture, and observability patterns learned from production systems.",
			},
		],
		["link", { rel: "icon", type: "image/png", href: "/images/specht.png" }],
	],

	bundler: viteBundler(),
	shouldPrefetch: false,

	plugins: [
		registerComponentsPlugin({
			componentsDir: path.resolve(__dirname, "./components"),
		}),
	],

	theme: plumeTheme({
		docsRepo: "https://github.com/SpechtLabs/golint-sl",
		docsDir: "doc",
		docsBranch: "main",

		editLink: true,
		lastUpdated: false,
		contributors: false,

		article: "/article/",

		cache: "filesystem",
		search: { provider: "local" },

		sidebar: {
			// Getting Started section - tutorials for newcomers
			"/getting-started/": [
				{
					text: "Getting Started",
					icon: "mdi:rocket-launch",
					prefix: "/getting-started/",
					items: [
						{ text: "Overview", link: "overview", icon: "mdi:eye" },
						{
							text: "Installation",
							link: "installation",
							icon: "mdi:download",
						},
						{
							text: "Quick Start",
							link: "quick",
							icon: "mdi:flash",
							badge: "5 min",
						},
					],
				},
			],

			// Guides section - how-to guides for specific tasks
			"/guides/": [
				{
					text: "How-to Guides",
					icon: "mdi:compass",
					prefix: "/guides/",
					items: [
						{
							text: "Integration",
							icon: "mdi:puzzle",
							items: [
								{
									text: "GitHub Actions",
									link: "github-actions",
									icon: "mdi:github",
								},
								{
									text: "Pre-commit Hooks",
									link: "pre-commit",
									icon: "mdi:hook",
								},
								{
									text: "golangci-lint",
									link: "golangci-lint",
									icon: "mdi:tools",
								},
							],
						},
						{
							text: "Configuration",
							icon: "mdi:cog",
							items: [
								{
									text: "Configure Analyzers",
									link: "configure-analyzers",
									icon: "mdi:tune",
								},
								{
									text: "Disable Analyzers",
									link: "disable-analyzers",
									icon: "mdi:toggle-switch-off",
								},
							],
						},
					],
				},
			],

			// Understanding section - explanation of concepts
			"/understanding/": [
				{
					text: "Understanding golint-sl",
					icon: "mdi:lightbulb",
					prefix: "/understanding/",
					items: [
						{ text: "Philosophy", link: "philosophy", icon: "mdi:brain" },
						{
							text: "Analyzer Categories",
							link: "categories",
							icon: "mdi:folder-multiple",
						},
						{
							text: "Wide Events Pattern",
							link: "wide-events",
							icon: "mdi:chart-timeline",
						},
						{
							text: "Kubernetes Patterns",
							link: "kubernetes-patterns",
							icon: "mdi:kubernetes",
						},
					],
				},
			],

			// Reference section - comprehensive reference material
			"/reference/": [
				{
					text: "Reference",
					icon: "mdi:book",
					prefix: "/reference/",
					items: [
						{ text: "CLI Reference", link: "cli", icon: "mdi:terminal" },
						{
							text: "Configuration",
							link: "configuration",
							icon: "mdi:file-cog",
						},
					],
				},
				{
					text: "Analyzers",
					icon: "mdi:magnify-scan",
					collapsed: true,
					prefix: "/reference/analyzers/",
					items: [
						{
							text: "Error Handling",
							icon: "mdi:alert-circle",
							collapsed: false,
							items: [
								{ text: "humaneerror", link: "humaneerror" },
								{ text: "errorwrap", link: "errorwrap" },
								{ text: "sentinelerrors", link: "sentinelerrors" },
							],
						},
						{
							text: "Observability",
							icon: "mdi:chart-line",
							collapsed: false,
							items: [
								{ text: "wideevents", link: "wideevents" },
								{ text: "contextlogger", link: "contextlogger" },
								{ text: "contextpropagation", link: "contextpropagation" },
							],
						},
						{
							text: "Kubernetes",
							icon: "mdi:kubernetes",
							collapsed: false,
							items: [
								{ text: "reconciler", link: "reconciler" },
								{ text: "statusupdate", link: "statusupdate" },
								{ text: "sideeffects", link: "sideeffects" },
							],
						},
						{
							text: "Testability",
							icon: "mdi:test-tube",
							collapsed: false,
							items: [
								{ text: "clockinterface", link: "clockinterface" },
								{ text: "interfaceconsistency", link: "interfaceconsistency" },
								{ text: "mockverify", link: "mockverify" },
								{ text: "optionspattern", link: "optionspattern" },
							],
						},
						{
							text: "Resources",
							icon: "mdi:memory",
							collapsed: false,
							items: [
								{ text: "resourceclose", link: "resourceclose" },
								{ text: "httpclient", link: "httpclient" },
							],
						},
						{
							text: "Safety",
							icon: "mdi:shield-check",
							collapsed: false,
							items: [
								{ text: "goroutineleak", link: "goroutineleak" },
								{ text: "nilcheck", link: "nilcheck" },
								{ text: "nopanic", link: "nopanic" },
								{ text: "nestingdepth", link: "nestingdepth" },
								{ text: "syncaccess", link: "syncaccess" },
							],
						},
						{
							text: "Clean Code",
							icon: "mdi:broom",
							collapsed: false,
							items: [
								{ text: "varscope", link: "varscope" },
								{ text: "closurecomplexity", link: "closurecomplexity" },
								{ text: "emptyinterface", link: "emptyinterface" },
								{ text: "returninterface", link: "returninterface" },
							],
						},
						{
							text: "Architecture",
							icon: "mdi:sitemap",
							collapsed: false,
							items: [
								{ text: "contextfirst", link: "contextfirst" },
								{ text: "pkgnaming", link: "pkgnaming" },
								{ text: "functionsize", link: "functionsize" },
								{ text: "exporteddoc", link: "exporteddoc" },
								{ text: "todotracker", link: "todotracker" },
								{ text: "hardcodedcreds", link: "hardcodedcreds" },
								{ text: "lifecycle", link: "lifecycle" },
								{ text: "dataflow", link: "dataflow" },
							],
						},
					],
				},
			],
		},

		markdown: {
			collapse: true,
			timeline: true,
			plot: true,
			mermaid: true,
			image: {
				figure: true,
				lazyload: true,
				mark: true,
				size: true,
			},
		},

		watermark: false,
	}),
});
