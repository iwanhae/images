<script lang="ts">
	import type { PageData } from './$types';
	let props: { data: PageData } = $props();

	// Access the dir parameter directly from the page store
	const currentDir = props.data.dir;

	// save the columns state to local storage
	$effect(() => {
		localStorage.setItem(`columns-page`, columns.toString());
	});

	let columns = $state(parseInt(localStorage.getItem(`columns-page`) || '5'));

	const data = (async () => {
		return await fetch(`/dir/${currentDir}`)
			.then((res) => res.json())
			.then((data) => data as string[])
			.then((data) => data.map((key) => ({ key, dir: currentDir })));
	})();
</script>

<div class="flex h-10 gap-2">
	<span>Columns: </span>
	{#each [1, 2, 3, 4, 5, 6] as i}
		<button class="h-7 w-10 rounded-md bg-gray-200 hover:bg-gray-300" onclick={() => (columns = i)}>
			{i}
		</button>
		<pre class="hidden grid-cols-{i}"></pre>
	{/each}
</div>

{#await data}
	<div class="h-96 w-60 rounded-md bg-gray-200">Loading...</div>
{:then items}
	<div class="grid grid-cols-{columns} gap-1">
		{#each items as item}
			<div class="min-h-32 bg-gray-200">
				<a href={`/obj/${item.key}`}>
					<img
						class="h-full w-full rounded-md object-cover"
						src={`/obj/${item.key}`}
						alt={item.key}
					/>
				</a>
			</div>
		{/each}
		<!-- Go back to the parent directory -->
		<a href={`/`}>
			<div class="flex min-h-32 items-center justify-center bg-gray-200">
				<span>Back</span>
			</div>
		</a>
	</div>
{/await}
