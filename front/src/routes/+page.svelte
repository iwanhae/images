<script lang="ts">
	import type { Metadata } from '$lib/types';

	// save the columns state to local storage
	$effect(() => {
		localStorage.setItem('columns-home', columns.toString());
	});

	let columns = $state(parseInt(localStorage.getItem('columns-home') || '5'));

	const data = (async () => {
		return await fetch('/random')
			.then((res) => res.json())
			.then((data) => data as Metadata[]);
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
			<div class="bg-gray-200">
				<a href={`/${item.dir}`}>
					<img
						class="h-full w-full rounded-md object-cover"
						src={`/obj/${item.key}`}
						alt={item.key}
					/>
				</a>
			</div>
		{/each}
		<!-- Refresh the page -->
		<button
			class="flex min-h-32 items-center justify-center bg-gray-200"
			onclick={() => location.reload()}
		>
			<span>Refresh</span>
		</button>
	</div>
{/await}
